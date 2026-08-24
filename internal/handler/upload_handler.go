package handler

import (
	"errors"
	"net/http"

	"gozaq/pkg/response"
	"gozaq/pkg/storage"
)

type UploadHandler struct {
	storage *storage.Storage
}

func NewUploadHandler(storage *storage.Storage) *UploadHandler {
	return &UploadHandler{
		storage: storage,
	}
}

func (h *UploadHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 10MB memory limit)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
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

	fileInfo, err := h.storage.SaveFile(header)
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
