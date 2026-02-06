package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SupabaseStorage handles file uploads to Supabase Storage
type SupabaseStorage struct {
	projectURL string
	apiKey     string
	bucketName string
	client     *http.Client
}

// NewSupabaseStorage creates a new Supabase storage client
func NewSupabaseStorage(projectURL, apiKey, bucketName string) *SupabaseStorage {
	// Clean the API key - remove any newlines, carriage returns, or other control characters
	cleanKey := strings.TrimSpace(apiKey)
	cleanKey = strings.ReplaceAll(cleanKey, "\n", "")
	cleanKey = strings.ReplaceAll(cleanKey, "\r", "")
	cleanKey = strings.ReplaceAll(cleanKey, "\t", "")

	return &SupabaseStorage{
		projectURL: strings.TrimSpace(strings.TrimSuffix(projectURL, "/")),
		apiKey:     cleanKey,
		bucketName: bucketName,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// UploadImage uploads an image to Supabase storage and returns the public URL
func (s *SupabaseStorage) UploadImage(ctx context.Context, path string, data []byte, contentType string) (string, error) {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.projectURL, s.bucketName, path)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-upsert", "true") // Overwrite if exists

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return s.GetPublicURL(path), nil
}

// FileExists checks if a file exists in the storage bucket
func (s *SupabaseStorage) FileExists(ctx context.Context, path string) (bool, error) {
	url := fmt.Sprintf("%s/storage/v1/object/info/%s/%s", s.projectURL, s.bucketName, path)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check file: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// GetPublicURL returns the public URL for a file path
func (s *SupabaseStorage) GetPublicURL(path string) string {
	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.projectURL, s.bucketName, path)
}
