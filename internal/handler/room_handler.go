package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"gozaq/internal/domain"
	"gozaq/internal/service"
	"gozaq/pkg/response"
	"gozaq/pkg/validator"
)

type RoomHandler struct {
	roomService *service.RoomService
}

func NewRoomHandler(roomService *service.RoomService) *RoomHandler {
	return &RoomHandler{roomService: roomService}
}

func (h *RoomHandler) ListRooms(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.roomService.ListRooms(r.Context())
	if err != nil {
		response.InternalServerError(w, "Failed to retrieve rooms", err.Error())
		return
	}
	response.OK(w, "Rooms retrieved successfully", rooms)
}

func (h *RoomHandler) GetRoom(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	room, err := h.roomService.GetRoom(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(w, "Room not found")
			return
		}
		response.InternalServerError(w, "Failed to retrieve room", err.Error())
		return
	}
	response.OK(w, "Room retrieved successfully", room)
}

func (h *RoomHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON payload", err.Error())
		return
	}

	if errs := validator.ValidateStruct(req); len(errs) > 0 {
		response.UnprocessableEntity(w, "Validation failed", errs)
		return
	}

	room, err := h.roomService.CreateRoom(r.Context(), &req)
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			response.Conflict(w, "Room with this ID or slug already exists")
			return
		}
		response.InternalServerError(w, "Failed to create room", err.Error())
		return
	}

	response.Created(w, "Room created successfully", room)
}

func (h *RoomHandler) UpdateRoom(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req domain.UpdateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON payload", err.Error())
		return
	}

	room, err := h.roomService.UpdateRoom(r.Context(), id, &req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(w, "Room not found")
			return
		}
		response.InternalServerError(w, "Failed to update room", err.Error())
		return
	}

	response.OK(w, "Room updated successfully", room)
}

func (h *RoomHandler) DeleteRoom(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.roomService.DeleteRoom(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(w, "Room not found")
			return
		}
		response.InternalServerError(w, "Failed to delete room", err.Error())
		return
	}

	response.OK(w, "Room deleted successfully", nil)
}
