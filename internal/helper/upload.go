package helper

import "strings"

// MaxUploadFileSizeBytes caps any single raw file upload (media, template samples) at 15MB.
const MaxUploadFileSizeBytes = 15 * 1024 * 1024

var allowedDocumentUploadContentTypes = map[string]bool{
	"application/pdf":               true,
	"application/msword":            true,
	"application/vnd.ms-excel":      true,
	"application/vnd.ms-powerpoint": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"text/plain": true,
}

// IsAllowedUploadContentType restricts uploads to images, videos, audio and common office/PDF documents.
func IsAllowedUploadContentType(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if strings.HasPrefix(contentType, "image/") || strings.HasPrefix(contentType, "video/") || strings.HasPrefix(contentType, "audio/") {
		return true
	}
	return allowedDocumentUploadContentTypes[contentType]
}
