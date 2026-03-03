package storage

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Storage handles file uploads to S3-compatible storage (Supabase Storage S3 protocol)
type S3Storage struct {
	client     *s3.Client
	bucketName string
	publicURL  string // Base URL for public access
}

// NewS3Storage creates a new S3-compatible storage client for Supabase
func NewS3Storage(endpoint, accessKey, secretKey, region, bucketName, publicURL string) (*S3Storage, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(endpoint)
	})

	return &S3Storage{
		client:     client,
		bucketName: bucketName,
		publicURL:  strings.TrimSuffix(publicURL, "/"),
	}, nil
}

// UploadImage uploads an image to S3 storage and returns the public URL
func (s *S3Storage) UploadImage(ctx context.Context, path string, data []byte, contentType string) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(path),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	return s.GetPublicURL(path), nil
}

// GetPublicURL returns the public URL for a file path
func (s *S3Storage) GetPublicURL(path string) string {
	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.publicURL, s.bucketName, path)
}

// FileExists checks if a file exists in the bucket
func (s *S3Storage) FileExists(ctx context.Context, path string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(path),
	})
	if err != nil {
		return false, nil // Assume not exists on error
	}
	return true, nil
}
