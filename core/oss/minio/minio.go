package minio

import (
	"context"
	"fmt"
	"math"
	"mime"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"golang.org/x/sync/errgroup"
)

// Minio is a wrapper around minio-go/v7 package.
// https://min.io/docs/minio/linux/reference/s3-api-compatibility.html

const (
	minPartSize = 5 * 1024 * 1024
	maxPartSize = 5 * 1024 * 1024 * 1024
)

type Minio struct {
	endpoint        string
	accessKeyID     string
	secretAccessKey string
	useSSL          bool
	maxParts        int
	semaphore       int

	Core *minio.Core
}

type Option func(*Minio)

func WithUseSSL(useSSL bool) Option {
	return func(m *Minio) {
		m.useSSL = useSSL
	}
}

func WithMaxParts(maxParts int) Option {
	return func(m *Minio) {
		m.maxParts = maxParts
	}
}

func NewMinio(endpoint, accessKeyID, secretAccessKey string, opts ...Option) (*Minio, error) {
	m := &Minio{
		endpoint:        endpoint,
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		useSSL:          true,
		maxParts:        10000,
		semaphore:       10,
	}

	for _, opt := range opts {
		opt(m)
	}

	var err error
	m.Core, err = minio.NewCore(m.endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(m.accessKeyID, m.secretAccessKey, ""),
		Secure: m.useSSL,
	})
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (m *Minio) CreateBucket(ctx context.Context, bucketName string) error {
	if exists, err := m.Core.BucketExists(ctx, bucketName); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("bucket %s already exists", bucketName)
	}

	return m.Core.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
}

func (m *Minio) DeleteBucket(ctx context.Context, bucketName string) error {
	if exists, err := m.Core.BucketExists(ctx, bucketName); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("bucket %s does not exist", bucketName)
	}

	return m.Core.RemoveBucket(ctx, bucketName)
}

func (m *Minio) ListIncompleteUploads(ctx context.Context, params *CreateMultipartUploadInput) (ListIncompleteUploadsOutput, error) {
	var output ListIncompleteUploadsOutput

	objectMultipartInfo := m.Core.ListIncompleteUploads(ctx, params.Bucket, params.Object, true)

	for info := range objectMultipartInfo {
		if info.Err != nil {
			return output, info.Err
		}
		if info.Key == params.Object {
			output.UploadId = info.UploadID

			parts, err := m.Core.ListObjectParts(ctx, params.Bucket, params.Object, info.UploadID, 0, m.maxParts)
			if err != nil {
				return output, err
			}

			output.Uploads = make(map[int]struct{}, len(parts.ObjectParts))
			for _, part := range parts.ObjectParts {
				output.Uploads[part.PartNumber] = struct{}{}
			}
		}
	}

	return output, nil
}

func (m *Minio) Upload(ctx context.Context, params *UploadInput) (UploadOutput, error) {
	var output UploadOutput

	if _, err := m.Core.StatObject(ctx, params.Bucket, params.Object, minio.StatObjectOptions{}); err == nil {
		return UploadOutput{Uploaded: true}, nil
	}

	presignUrl, err := m.Core.PresignedPutObject(ctx, params.Bucket, params.Object, time.Duration(params.Expires)*time.Second)
	if err != nil {
		return output, err
	}

	output.Url = presignUrl

	return output, nil
}

func (m *Minio) CreateMultipartUpload(ctx context.Context, params *CreateMultipartUploadInput) (CreateMultipartUploadOutput, error) {
	var output CreateMultipartUploadOutput

	if params.PartSize < minPartSize || params.PartSize > maxPartSize {
		return output, fmt.Errorf("part size must be between %d and %d bytes", minPartSize, maxPartSize)
	}

	if _, err := m.Core.StatObject(ctx, params.Bucket, params.Object, minio.StatObjectOptions{}); err == nil {
		return CreateMultipartUploadOutput{Uploaded: true}, nil
	}

	listIncompleteUploads, err := m.ListIncompleteUploads(ctx, params)
	if err != nil {
		return output, err
	}

	uploadId := listIncompleteUploads.UploadId
	if uploadId == "" {
		uploadId, err = m.Core.NewMultipartUpload(ctx, params.Bucket, params.Object, minio.PutObjectOptions{
			ContentType: m.contextType(params.Object),
		})
		if err != nil {
			return output, err
		}
	}

	var eg errgroup.Group
	partInfos := m.sharding(params.ObjectSize, params.PartSize)
	partInfoChan := make(chan PartInfo, len(partInfos))

	sem := make(chan struct{}, m.semaphore) // semaphore to limit the number of concurrent goroutines
	for _, partInfo := range partInfos {
		if _, ok := listIncompleteUploads.Uploads[partInfo.PartNumber]; ok {
			continue
		}

		partInfo := partInfo // capture range variable
		eg.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()

			urlValues := url.Values{
				"uploadId":   {uploadId},
				"partNumber": {strconv.Itoa(partInfo.PartNumber)},
			}

			partUrl, err := m.Core.Presign(ctx, http.MethodPut, params.Bucket, params.Object, time.Duration(params.Expires)*time.Second, urlValues)
			if err != nil {
				return err
			}

			partInfoChan <- PartInfo{
				PartNumber: partInfo.PartNumber,
				BeginSize:  partInfo.BeginSize,
				EndSize:    partInfo.EndSize,
				Url:        partUrl,
			}

			return nil
		})
	}

	go func() {
		eg.Wait()
		close(partInfoChan)
	}()

	for partInfo := range partInfoChan {
		output.PartInfos = append(output.PartInfos, partInfo)
	}

	if err := eg.Wait(); err != nil {
		return output, err
	}

	output.UploadId = uploadId

	return output, nil
}

func (m *Minio) CompleteMultipartUpload(ctx context.Context, params *CompleteMultipartUploadInput) (minio.UploadInfo, error) {
	result, err := m.Core.ListObjectParts(ctx, params.Bucket, params.Object, params.UploadId, 0, m.maxParts)
	if err != nil {
		return minio.UploadInfo{}, err
	}

	if len(result.ObjectParts) == 0 {
		return minio.UploadInfo{}, nil
	}

	parts := make([]minio.CompletePart, len(result.ObjectParts))
	for i, part := range result.ObjectParts {
		parts[i] = minio.CompletePart{
			PartNumber: part.PartNumber,
			ETag:       part.ETag,
		}
	}

	sort.Slice(parts, func(i, j int) bool {
		return parts[i].PartNumber < parts[j].PartNumber
	})

	return m.Core.CompleteMultipartUpload(ctx, params.Bucket, params.Object, params.UploadId, parts, minio.PutObjectOptions{
		ContentType: m.contextType(params.Object),
	})
}

func (m *Minio) AbortMultipartUpload(ctx context.Context, params *AbortMultipartUploadInput) error {
	return m.Core.AbortMultipartUpload(ctx, params.Bucket, params.Object, params.UploadId)
}

// contextType returns the content type of an object
func (m *Minio) contextType(objectName string) string {
	if contentType := mime.TypeByExtension(path.Ext(objectName)); contentType != "" {
		return contentType
	}
	return "application/octet-stream"
}

// sharding splits an object into parts
func (m *Minio) sharding(objectSize, partSize uint64) []PartInfo {
	partCount := int(math.Ceil(float64(objectSize) / float64(partSize)))
	partInfos := make([]PartInfo, 0, partCount)

	for i := 0; i < partCount; i++ {
		beginSize := uint64(i) * partSize
		endSize := beginSize + partSize
		if endSize > objectSize {
			endSize = objectSize
		}

		partInfos = append(partInfos, PartInfo{
			PartNumber: i + 1,
			BeginSize:  beginSize,
			EndSize:    endSize,
		})
	}

	return partInfos
}
