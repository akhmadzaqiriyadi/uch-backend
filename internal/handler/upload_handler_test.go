package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"gozaq/pkg/storage"
)

type mockStorage struct {
	saveFunc   func(header *multipart.FileHeader) (*storage.FileInfo, error)
	deleteFunc func(filename string) error
	getURLFunc func(filename string) string
}

func (m *mockStorage) SaveFile(header *multipart.FileHeader) (*storage.FileInfo, error) {
	if m.saveFunc != nil {
		return m.saveFunc(header)
	}
	return &storage.FileInfo{
		OriginalName: header.Filename,
		FileName:     "test-uuid.png",
		URL:          "http://localhost:8080/uploads/test-uuid.png",
		MimeType:     "image/png",
	}, nil
}

func (m *mockStorage) DeleteFile(filename string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(filename)
	}
	return nil
}

func (m *mockStorage) GetURL(filename string) string {
	if m.getURLFunc != nil {
		return m.getURLFunc(filename)
	}
	return "http://localhost:8080/uploads/" + filename
}

func TestUploadHandler_UploadFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockStore := &mockStorage{}
		h := NewUploadHandler(mockStore)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "avatar.png")
		_, _ = part.Write([]byte("fake-png-data"))
		_ = writer.Close()

		req := httptest.NewRequest("POST", "/uploads", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		h.UploadFile(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "File uploaded successfully")
	})

	t.Run("Missing File Field", func(t *testing.T) {
		mockStore := &mockStorage{}
		h := NewUploadHandler(mockStore)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.Close()

		req := httptest.NewRequest("POST", "/uploads", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		h.UploadFile(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("File Too Large", func(t *testing.T) {
		mockStore := &mockStorage{
			saveFunc: func(header *multipart.FileHeader) (*storage.FileInfo, error) {
				return nil, storage.ErrFileTooLarge
			},
		}
		h := NewUploadHandler(mockStore)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "big.png")
		_, _ = part.Write([]byte("huge"))
		_ = writer.Close()

		req := httptest.NewRequest("POST", "/uploads", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		h.UploadFile(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "File size exceeds 5MB limit")
	})

	t.Run("Invalid MIME type", func(t *testing.T) {
		mockStore := &mockStorage{
			saveFunc: func(header *multipart.FileHeader) (*storage.FileInfo, error) {
				return nil, storage.ErrInvalidFileType
			},
		}
		h := NewUploadHandler(mockStore)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "bad.exe")
		_, _ = part.Write([]byte("bad"))
		_ = writer.Close()

		req := httptest.NewRequest("POST", "/uploads", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		h.UploadFile(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}
