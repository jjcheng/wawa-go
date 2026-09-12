package service

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/types"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type File struct {
	logger     *Logger
	client     *oss.Client
	bucket     *oss.Bucket
	bucketName string
	endpoint   string
}

func NewFileService(logger *Logger) *File {
	config := cfg.Default().AliyunOSS
	client, err := oss.New(config.Endpoint, config.AccessKeyID, config.AccessKeySecret)
	if err != nil {
		panic(fmt.Errorf("failed to create oss client: %w", err))
	}
	bucket, err := client.Bucket(config.BucketName)
	if err != nil {
		panic(fmt.Errorf("failed to get oss bucket: %w", err))
	}
	f := &File{
		logger:     logger,
		client:     client,
		bucket:     bucket,
		bucketName: config.BucketName,
		endpoint:   config.Endpoint,
	}
	return f
}

// UploadFile uploads a file to Aliyun OSS with a permanent public URL
// Returns a simple public URL (requires bucket to have public reading enabled)
// storageClass can be types.StorageClassStandard, types.StorageClassIA, types.StorageClassArchive
func (f *File) UploadFile(data []byte, folderName string, filename string, storageClass types.StorageClass) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("file data cannot be empty")
	}
	// Generate unique blob name
	blobName, err := f.generateBlobName(folderName, filename)
	if err != nil {
		return "", fmt.Errorf("failed to generate blob name: %w", err)
	}
	var ossStorageClass oss.StorageClassType
	switch storageClass {
	case types.StorageClassCool:
		ossStorageClass = oss.StorageIA
	case types.StorageClassArchive:
		ossStorageClass = oss.StorageArchive
	case types.StorageClassCold:
		ossStorageClass = oss.StorageColdArchive
	default:
		ossStorageClass = oss.StorageStandard
	}
	options := []oss.Option{
		oss.ObjectStorageClass(ossStorageClass),
		oss.Meta("filename", filename),
		oss.Meta("permanent", "true"), // marker to indicate this is a permanent file
	}
	// Upload to OSS
	err = f.bucket.PutObject(blobName, bytes.NewReader(data), options...)
	if err != nil {
		f.logger.ErrorFunction(err, blobName, options)
		return "", fmt.Errorf("failed to upload blob: %w", err)
	}
	// construct public URL
	protocol := "https://"
	domain := f.endpoint
	if strings.HasPrefix(domain, "http://") {
		protocol = "http://"
		domain = strings.TrimPrefix(domain, "http://")
	} else if strings.HasPrefix(domain, "https://") {
		protocol = "https://"
		domain = strings.TrimPrefix(domain, "https://")
	}
	publicURL := fmt.Sprintf("%s%s.%s/%s", protocol, f.bucketName, domain, blobName)
	return publicURL, nil
}

// UploadTempFile uploads a file to Aliyun OSS with a very short lifetime
// Returns a signed URL that will expire after the specified duration
func (f *File) UploadTempFile(data []byte, folderName string, filename string, lifetime time.Duration) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("file data cannot be empty")
	}
	// Validate and adjust lifetime
	if lifetime <= 0 {
		lifetime = 10 * time.Minute
	}
	// Generate unique blob name
	blobName, err := f.generateBlobName(folderName, filename)
	if err != nil {
		return "", fmt.Errorf("failed to generate blob name: %w", err)
	}
	expiresAt := time.Now().Add(lifetime)
	expires := expiresAt.Format(time.RFC3339)
	options := []oss.Option{
		oss.Meta("filename", filename),
		oss.Meta("expires_at", expires),
	}
	err = f.bucket.PutObject(blobName, bytes.NewReader(data), options...)
	if err != nil {
		return "", fmt.Errorf("failed to upload blob: %w", err)
	}
	// Generate Signed URL with expiration
	signedURL, err := f.bucket.SignURL(blobName, oss.HTTPGet, int64(lifetime.Seconds()))
	if err != nil {
		// If URL generation fails, delete the uploaded blob
		_ = f.deleteBlob(blobName)
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}
	f.logger.Infof("File uploaded to OSS: %s, Size=%d bytes, Expires=%s", blobName, len(data), expiresAt.Format(time.RFC3339))
	return signedURL, nil
}

// GetFile retrieves file metadata and content
func (f *File) GetFile(blobName string) ([]byte, string, error) {
	props, err := f.bucket.GetObjectMeta(blobName)
	if err != nil {
		return nil, "", fmt.Errorf("blob not found: %w", err)
	}
	// Check expiration
	if expiresAtStr := props.Get("X-Oss-Meta-Expires_at"); expiresAtStr != "" {
		expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
		if err == nil && time.Now().After(expiresAt) {
			// Delete expired blob
			_ = f.deleteBlob(blobName)
			return nil, "", fmt.Errorf("file has expired")
		}
	}
	// Download blob
	body, err := f.bucket.GetObject(blobName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download blob: %w", err)
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read blob data: %w", err)
	}

	filename := props.Get("X-Oss-Meta-Filename")
	return data, filename, nil
}

// DeleteFile manually deletes a blob before its expiration
func (f *File) DeleteFile(blobName string) error {
	return f.deleteBlob(blobName)
}

// deleteBlob is an internal helper to delete a blob
func (f *File) deleteBlob(blobName string) error {
	err := f.bucket.DeleteObject(blobName)
	if err != nil {
		return fmt.Errorf("failed to delete blob: %w", err)
	}
	f.logger.Infof("Blob deleted: %s", blobName)
	return nil
}

// generateBlobName creates a blob name using folder and filename
func (f *File) generateBlobName(folderName string, filename string) (string, error) {
	// Strip any path from filename
	base := filename
	lastSlash := strings.LastIndex(filename, "/")
	if lastSlash != -1 {
		base = filename[lastSlash+1:]
	}
	if folderName != "" {
		folderName = strings.TrimSuffix(folderName, "/")
		return fmt.Sprintf("%s/%s", folderName, base), nil
	}
	return base, nil
}
