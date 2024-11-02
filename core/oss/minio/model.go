package minio

import "net/url"

type UploadInput struct {
	Bucket  string `json:"bucket"`
	Object  string `json:"object"`
	Expires uint64 `json:"expires"`
}

type UploadOutput struct {
	Uploaded bool     `json:"uploaded"`
	Url      *url.URL `json:"url"`
}

type CreateMultipartUploadInput struct {
	Bucket     string `json:"bucket"`
	Object     string `json:"object"`
	Expires    uint64 `json:"expires"`
	ObjectSize uint64 `json:"object_size"`
	PartSize   uint64 `json:"part_size"`
}

type CreateMultipartUploadOutput struct {
	Uploaded  bool       `json:"uploaded"`
	UploadId  string     `json:"upload_id"`
	PartSize  uint64     `json:"part_size"`
	PartInfos []PartInfo `json:"part_infos"`
}

type CompleteMultipartUploadInput struct {
	UploadId string `json:"upload_id"`
	Bucket   string `json:"bucket"`
	Object   string `json:"object"`
}

type AbortMultipartUploadInput struct {
	UploadId string `json:"upload_id"`
	Bucket   string `json:"bucket"`
	Object   string `json:"object"`
}

type PartInfo struct {
	PartNumber int      `json:"part_number"`
	BeginSize  uint64   `json:"begin_size"`
	EndSize    uint64   `json:"end_size"`
	Url        *url.URL `json:"url"`
}

type ListIncompleteUploadsOutput struct {
	UploadId string           `json:"upload_id"`
	Uploads  map[int]struct{} `json:"uploads"`
}
