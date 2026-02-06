package storage

import "context"

// Storage defines the interface for file storage operations
type Storage interface {
	UploadImage(ctx context.Context, path string, data []byte, contentType string) (string, error)
	GetPublicURL(path string) string
	FileExists(ctx context.Context, path string) (bool, error)
}
