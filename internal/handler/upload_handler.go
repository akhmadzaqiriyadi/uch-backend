package handler

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"gozaq/pkg/response"
	"gozaq/pkg/storage"
)

type UploadHandler struct {
	storage storage.Storage
}

func NewUploadHandler(storage storage.Storage) *UploadHandler {
	return &UploadHandler{
		storage: storage,
	}
}

func (h *UploadHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 25MB memory limit)
	if err := r.ParseMultipartForm(25 << 20); err != nil {
		response.BadRequest(w, "Failed to parse multipart form data", err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "Field 'file' is required in form-data", nil)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	folder := strings.TrimSpace(r.FormValue("folder"))
	if folder == "" {
		folder = "rooms"
	}

	fileInfo, err := h.storage.SaveFileWithFolder(header, folder)
	if err != nil {
		if errors.Is(err, storage.ErrFileTooLarge) {
			response.BadRequest(w, "File size exceeds 5MB limit", nil)
			return
		}
		if errors.Is(err, storage.ErrInvalidFileType) {
			response.UnprocessableEntity(w, "Unsupported file format. Allowed: JPEG, PNG, WebP, GIF, PDF, JSON", err.Error())
			return
		}
		response.InternalServerError(w, "Failed to store uploaded file", err.Error())
		return
	}

	response.Created(w, "File uploaded successfully", fileInfo)
}

func (h *UploadHandler) ServeFile(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "*")
	if key == "" {
		key = strings.TrimPrefix(r.URL.Path, "/uploads/")
		key = strings.TrimPrefix(key, "/api/v1/upload/file/")
	}
	key = strings.TrimPrefix(key, "/")

	if key == "" {
		response.NotFound(w, "File key is required")
		return
	}

	reader, mimeType, err := h.storage.GetFile(r.Context(), key)
	if err != nil {
		response.NotFound(w, "File not found in storage")
		return
	}
	defer func() {
		_ = reader.Close()
	}()

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Disposition", "inline")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}
