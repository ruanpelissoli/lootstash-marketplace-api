package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockStorage is a mock implementation of storage.Storage
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) UploadImage(ctx context.Context, path string, data []byte, contentType string) (string, error) {
	args := m.Called(ctx, path, data, contentType)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) GetPublicURL(path string) string {
	args := m.Called(path)
	return args.String(0)
}

func (m *MockStorage) FileExists(ctx context.Context, path string) (bool, error) {
	args := m.Called(ctx, path)
	return args.Bool(0), args.Error(1)
}
