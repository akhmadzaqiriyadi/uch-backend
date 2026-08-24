package storage

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gozaq/config"
)

func createTestFileHeader(t *testing.T, fieldName, fileName, content string) *multipart.FileHeader {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, fileName)
	require.NoError(t, err)
	_, err = part.Write([]byte(content))
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	err = req.ParseMultipartForm(10 << 20)
	require.NoError(t, err)

	_, header, err := req.FormFile(fieldName)
	require.NoError(t, err)
	return header
}

func TestLocalStorage_SaveFile_Success(t *testing.T) {
	tempDir := t.TempDir()
	store := NewLocalStorage(tempDir, "http://localhost:8080/uploads", 5<<20)

	// Valid PNG content
	pngHeader := "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15c4"
	header := createTestFileHeader(t, "file", "test.png", pngHeader)

	info, err := store.SaveFile(header)
	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, "test.png", info.OriginalName)
	assert.NotEmpty(t, info.FileName)
	assert.Contains(t, info.URL, "http://localhost:8080/uploads/")
	assert.Equal(t, "image/png", info.MimeType)
	assert.FileExists(t, info.FilePath)

	// Test GetURL
	assert.Equal(t, "http://localhost:8080/uploads/"+info.FileName, store.GetURL(info.FileName))

	// Test DeleteFile
	err = store.DeleteFile(info.FileName)
	require.NoError(t, err)
	assert.NoFileExists(t, info.FilePath)
}

func TestLocalStorage_SaveFile_TooLarge(t *testing.T) {
	tempDir := t.TempDir()
	// Limit to 10 bytes
	store := NewLocalStorage(tempDir, "http://localhost:8080/uploads", 10)

	header := createTestFileHeader(t, "file", "large.json", `{"data":"this is definitely longer than 10 bytes"}`)

	info, err := store.SaveFile(header)
	assert.Nil(t, info)
	assert.ErrorIs(t, err, ErrFileTooLarge)
}

func TestLocalStorage_SaveFile_InvalidMime(t *testing.T) {
	tempDir := t.TempDir()
	store := NewLocalStorage(tempDir, "http://localhost:8080/uploads", 5<<20)

	// Executable or unsupported binary header
	header := createTestFileHeader(t, "file", "malicious.exe", "MZ\x90\x00\x03\x00\x00\x00")

	info, err := store.SaveFile(header)
	assert.Nil(t, info)
	assert.ErrorIs(t, err, ErrInvalidFileType)
}

func TestLocalStorage_DeleteFile_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	store := NewLocalStorage(tempDir, "http://localhost:8080/uploads", 5<<20)

	err := store.DeleteFile("non_existent_file.txt")
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestLocalStorage_GetURL(t *testing.T) {
	store := NewLocalStorage("/tmp", "http://cdn.example.com/files/", 5<<20)
	url := store.GetURL("avatar.png")
	assert.Equal(t, "http://cdn.example.com/files/avatar.png", url)
}

// MockS3Client provides mock implementations for S3ClientAPI
type MockS3Client struct {
	mock.Mock
}

func (m *MockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	args := m.Called(ctx, params)
	return &s3.PutObjectOutput{}, args.Error(0)
}

func (m *MockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	args := m.Called(ctx, params)
	return &s3.DeleteObjectOutput{}, args.Error(0)
}

func TestS3Storage_SaveFile_Success(t *testing.T) {
	mockClient := new(MockS3Client)
	mockClient.On("PutObject", mock.Anything, mock.AnythingOfType("*s3.PutObjectInput")).Return(nil)

	s3Store := NewS3StorageWithClient(mockClient, "my-test-bucket", "ap-southeast-1", "https://cdn.example.com", 5<<20)

	pngHeader := "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15c4"
	header := createTestFileHeader(t, "file", "avatar.png", pngHeader)

	info, err := s3Store.SaveFile(header)
	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, "avatar.png", info.OriginalName)
	assert.Contains(t, info.URL, "https://cdn.example.com/")
	assert.Equal(t, "image/png", info.MimeType)
	assert.Contains(t, info.FilePath, "s3://my-test-bucket/")

	mockClient.AssertExpectations(t)
}

func TestS3Storage_DeleteFile_Success(t *testing.T) {
	mockClient := new(MockS3Client)
	mockClient.On("DeleteObject", mock.Anything, mock.AnythingOfType("*s3.DeleteObjectInput")).Return(nil)

	s3Store := NewS3StorageWithClient(mockClient, "my-test-bucket", "ap-southeast-1", "", 5<<20)

	err := s3Store.DeleteFile("avatar.png")
	require.NoError(t, err)

	assert.Equal(t, "https://my-test-bucket.s3.ap-southeast-1.amazonaws.com/avatar.png", s3Store.GetURL("avatar.png"))
	mockClient.AssertExpectations(t)
}

func TestNewStorage_Factory(t *testing.T) {
	t.Run("Default to Local Storage", func(t *testing.T) {
		cfg := &config.Config{
			App: config.AppConfig{BaseURL: "http://localhost:8080"},
			Storage: config.StorageConfig{
				Driver:      "local",
				LocalDir:    "/tmp/test_uploads",
				MaxFileSize: 5 << 20,
			},
		}

		s, err := NewStorage(cfg)
		require.NoError(t, err)
		_, ok := s.(*LocalStorage)
		assert.True(t, ok)
	})

	t.Run("S3 Storage Driver Missing Bucket", func(t *testing.T) {
		cfg := &config.Config{
			Storage: config.StorageConfig{
				Driver:   "s3",
				S3Bucket: "",
			},
		}

		_, err := NewStorage(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "S3_BUCKET is required")
	})
}
