package minio

import (
	"context"
	"testing"
)

func TestCreateBucket(t *testing.T) {
	m, err := NewMinio("localhost:9000", "root", "12345678", WithUseSSL(false))
	if err != nil {
		t.Fatal(err)
	}

	if err := m.CreateBucket(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteBucket(t *testing.T) {
	m, err := NewMinio("localhost:9000", "root", "12345678", WithUseSSL(false))
	if err != nil {
		t.Fatal(err)
	}

	if err := m.DeleteBucket(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
}

func TestUpload(t *testing.T) {
	m, err := NewMinio("localhost:9000", "root", "12345678", WithUseSSL(false))
	if err != nil {
		t.Fatal(err)
	}

	params := &UploadInput{
		Bucket:  "test",
		Object:  "uTools-5.2.0.exe",
		Expires: 3600,
	}

	output, err := m.Upload(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(output)
}

func TestCreateMultipartUpload(t *testing.T) {
	m, err := NewMinio("localhost:9000", "root", "12345678", WithUseSSL(false))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(m.contextType("uTools-5.2.0.exe"))
	params := &CreateMultipartUploadInput{
		Bucket:     "test",
		Object:     "uTools-5.2.0.exe",
		ObjectSize: 68309368,
		PartSize:   52428800,
		Expires:    3600,
	}

	output, err := m.CreateMultipartUpload(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(output)
}

func TestCompleteMultipartUpload(t *testing.T) {
	m, err := NewMinio("localhost:9000", "root", "12345678", WithUseSSL(false))
	if err != nil {
		t.Fatal(err)
	}

	params := &CompleteMultipartUploadInput{
		UploadId: "M2FjZGQyMmEtODAxZi00Y2YyLWJlNjItZDRiOTc0NzBhM2FmLjg4ZjcwNDk3LTBmY2ItNDFhNC04NWVlLWEwZGZiMzg1ZGQ5MXgxNzMwNDkwNjIzODc0NDk3Mzg1",
		Bucket:   "test",
		Object:   "uTools-5.2.0.exe",
	}

	if _, err := m.CompleteMultipartUpload(context.Background(), params); err != nil {
		t.Fatal(err)
	}
}

func TestSetBucketPolicy(t *testing.T) {
	m, err := NewMinio("localhost:9000", "root", "12345678", WithUseSSL(false))
	if err != nil {
		t.Fatal(err)
	}

	policy := `{
        "Version": "2012-10-17",
        "Statement": [
            {
                "Effect": "Allow",
                "Principal": "*",
                "Action": [
                    "s3:GetObject"
                ],
                "Resource": [
                    "arn:aws:s3:::` + "test" + `/*"
                ]
            }
        ]
    }`

	if err := m.Core.SetBucketPolicy(context.Background(), "test", policy); err != nil {
		t.Fatal(err)
	}
}
