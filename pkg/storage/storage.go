package storage

import (
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
)

type FileInfo struct {
	OriginalName string `json:"original_name"`
	FileName     string `json:"file_name"`
	FilePath     string `json:"file_path"`
	URL          string `json:"url"`
	Size         int64  `json:"size"`
	MimeType     string `json:"mime_type"`
}

type Storage struct {
	baseDir      string
	baseURL      string
	maxSize      int64
	allowedMimes map[string]bool
}

func NewLocalStorage(baseDir, baseURL string, maxSizeBytes int64) *Storage {
	_ = os.MkdirAll(baseDir, 0755)

	allowedMimes := map[string]bool{
		"image/jpeg":       true,
		"image/png":        true,
		"image/webp":       true,
		"image/gif":        true,
		"application/pdf":  true,
		"application/json": true,
	}

	return &Storage{
		baseDir:      baseDir,
		baseURL:      baseURL,
		maxSize:      maxSizeBytes,
		allowedMimes: allowedMimes,
	}
}

func (s *Storage) SaveFile(header *multipart.FileHeader) (*FileInfo, error) {
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
	uniqueName := fmt.Sprintf("%s%s", uuid.New().String(), strings.ToLower(ext))
	targetPath := filepath.Join(s.baseDir, uniqueName)

	dst, err := os.Create(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		_ = dst.Close()
	}()

	written, err := io.Copy(dst, file)
	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	return &FileInfo{
		OriginalName: header.Filename,
		FileName:     uniqueName,
		FilePath:     targetPath,
		URL:          fmt.Sprintf("%s/%s", s.baseURL, uniqueName),
		Size:         written,
		MimeType:     mimeType,
	}, nil
}
