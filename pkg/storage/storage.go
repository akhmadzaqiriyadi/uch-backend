package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrFileTooLarge    = errors.New("file size exceeds maximum allowed limit")
	ErrInvalidFileType = errors.New("unsupported file type")
	ErrFileNotFound    = errors.New("file not found")
)

type FileInfo struct {
	OriginalName string `json:"original_name"`
	FileName     string `json:"file_name"`
	FilePath     string `json:"file_path"`
	URL          string `json:"url"`
	Size         int64  `json:"size"`
	MimeType     string `json:"mime_type"`
}

// Storage is the abstraction for file storage backends (Local, S3, MinIO, GCS)
type Storage interface {
	SaveFile(header *multipart.FileHeader) (*FileInfo, error)
	SaveFileWithFolder(header *multipart.FileHeader, folder string) (*FileInfo, error)
	DeleteFile(filename string) error
	GetURL(filename string) string
	GetFile(ctx context.Context, key string) (io.ReadCloser, string, error)
}

// LocalStorage implements Storage for the local file system
type LocalStorage struct {
	baseDir      string
	baseURL      string
	maxSize      int64
	allowedMimes map[string]bool
}

func NewLocalStorage(baseDir, baseURL string, maxSizeBytes int64) *LocalStorage {
	_ = os.MkdirAll(baseDir, 0755)

	allowedMimes := map[string]bool{
		"image/jpeg":       true,
		"image/png":        true,
		"image/webp":       true,
		"image/gif":        true,
		"application/pdf":  true,
		"application/json": true,
		"text/plain":       true,
	}

	return &LocalStorage{
		baseDir:      baseDir,
		baseURL:      baseURL,
		maxSize:      maxSizeBytes,
		allowedMimes: allowedMimes,
	}
}

func (s *LocalStorage) SaveFile(header *multipart.FileHeader) (*FileInfo, error) {
	return s.SaveFileWithFolder(header, "")
}

func (s *LocalStorage) SaveFileWithFolder(header *multipart.FileHeader, folder string) (*FileInfo, error) {
	if header.Size > s.maxSize {
		return nil, ErrFileTooLarge
	}

	file, err := header.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	// Sniff MIME type from first 512 bytes
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read file header: %w", err)
	}
	mimeType := http.DetectContentType(buffer[:n])

	// Strip parameters like charset from detected MIME
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}

	if !s.allowedMimes[mimeType] {
		return nil, fmt.Errorf("%w: %s", ErrInvalidFileType, mimeType)
	}

	// Seek back to start
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to reset file pointer: %w", err)
	}

	// Generate safe unique filename
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".bin"
	}
	keyName := fmt.Sprintf("%s%s", uuid.New().String(), strings.ToLower(ext))
	targetDir := s.baseDir
	if folder != "" {
		cleanFolder := strings.Trim(folder, "/")
		targetDir = filepath.Join(s.baseDir, cleanFolder)
		_ = os.MkdirAll(targetDir, 0755)
		keyName = fmt.Sprintf("%s/%s", cleanFolder, keyName)
	}

	targetPath := filepath.Join(s.baseDir, filepath.FromSlash(keyName))

	dst, err := os.Create(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		_ = dst.Close()
	}()

	written, err := io.Copy(dst, file)
	if err != nil {
		_ = os.Remove(targetPath)
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	return &FileInfo{
		OriginalName: header.Filename,
		FileName:     keyName,
		FilePath:     targetPath,
		URL:          s.GetURL(keyName),
		Size:         written,
		MimeType:     mimeType,
	}, nil
}

func (s *LocalStorage) DeleteFile(filename string) error {
	// Prevent directory traversal
	cleanName := filepath.Clean(filename)
	cleanName = strings.TrimPrefix(cleanName, "/")
	targetPath := filepath.Join(s.baseDir, filepath.FromSlash(cleanName))

	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return ErrFileNotFound
	}

	return os.Remove(targetPath)
}

func (s *LocalStorage) GetURL(filename string) string {
	cleanName := strings.TrimPrefix(filepath.ToSlash(filename), "/")
	return fmt.Sprintf("%s/%s", strings.TrimRight(s.baseURL, "/"), cleanName)
}

func (s *LocalStorage) GetFile(ctx context.Context, key string) (io.ReadCloser, string, error) {
	cleanPath := filepath.Clean(key)
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	targetPath := filepath.Join(s.baseDir, filepath.FromSlash(cleanPath))

	f, err := os.Open(targetPath)
	if err != nil {
		return nil, "", ErrFileNotFound
	}

	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	_, _ = f.Seek(0, io.SeekStart)
	mime := http.DetectContentType(buf[:n])
	if idx := strings.Index(mime, ";"); idx != -1 {
		mime = strings.TrimSpace(mime[:idx])
	}
	return f, mime, nil
}
