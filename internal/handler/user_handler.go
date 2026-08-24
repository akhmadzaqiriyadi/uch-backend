package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"gozaq/internal/domain"
	"gozaq/internal/handler/middleware"
	"gozaq/pkg/pagination"
	"gozaq/pkg/response"
	"gozaq/pkg/validator"
)

type UserHandler struct {
	userService domain.UserService
}

func NewUserHandler(userService domain.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON request payload", err.Error())
		return
	}

	if errs := validator.ValidateStruct(req); len(errs) > 0 {
		response.UnprocessableEntity(w, "Validation failed on request body", errs)
		return
	}

	authResp, err := h.userService.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			response.Conflict(w, "Email is already registered")
			return
		}
		response.InternalServerError(w, "Failed to register user", err.Error())
		return
	}

	response.Created(w, "User registered successfully", authResp)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON request payload", err.Error())
		return
	}

	if errs := validator.ValidateStruct(req); len(errs) > 0 {
		response.UnprocessableEntity(w, "Validation failed on request body", errs)
		return
	}

	authResp, err := h.userService.Login(r.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			response.Unauthorized(w, "Invalid email or password")
			return
		}
		response.InternalServerError(w, "Failed to login", err.Error())
		return
	}

	response.OK(w, "Login successful", authResp)
}

func (h *UserHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req domain.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON request payload", err.Error())
		return
	}

	if errs := validator.ValidateStruct(req); len(errs) > 0 {
		response.UnprocessableEntity(w, "Validation failed on request body", errs)
		return
	}

	authResp, err := h.userService.RefreshToken(r.Context(), req)
	if err != nil {
		response.Unauthorized(w, "Invalid or expired refresh token")
		return
	}

	response.OK(w, "Token refreshed successfully", authResp)
}

func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		response.Unauthorized(w, "Unauthorized")
		return
	}

	var req domain.RefreshTokenRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := h.userService.Logout(r.Context(), userID, req.RefreshToken); err != nil {
		response.InternalServerError(w, "Failed to logout", err.Error())
		return
	}

	response.OK(w, "Logged out successfully", nil)
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		response.Unauthorized(w, "Unauthorized")
		return
	}

	user, err := h.userService.GetProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(w, "User profile not found")
			return
		}
		response.InternalServerError(w, "Failed to retrieve profile", err.Error())
		return
	}

	response.OK(w, "Profile retrieved successfully", user)
}

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		response.Unauthorized(w, "Unauthorized")
		return
	}

	var req domain.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON request payload", err.Error())
		return
	}

	if errs := validator.ValidateStruct(req); len(errs) > 0 {
		response.UnprocessableEntity(w, "Validation failed on request body", errs)
		return
	}

	user, err := h.userService.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		response.InternalServerError(w, "Failed to update profile", err.Error())
		return
	}

	response.OK(w, "Profile updated successfully", user)
}

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	params := pagination.FromRequest(r)
	search := r.URL.Query().Get("search")

	res, err := h.userService.ListUsers(r.Context(), params, search)
	if err != nil {
		response.InternalServerError(w, "Failed to fetch users", err.Error())
		return
	}

	response.OK(w, "Users retrieved successfully", res)
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	targetID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "Invalid user ID UUID format", nil)
		return
	}

	if err := h.userService.DeleteUser(r.Context(), targetID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(w, "User not found")
			return
		}
		response.InternalServerError(w, "Failed to delete user", err.Error())
		return
	}

	response.OK(w, "User deleted successfully", nil)
}

func (h *UserHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	targetID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "Invalid user ID UUID format", nil)
		return
	}

	var req domain.UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON request payload", err.Error())
		return
	}

	if errs := validator.ValidateStruct(req); len(errs) > 0 {
		response.UnprocessableEntity(w, "Validation failed on request body", errs)
		return
	}

	user, err := h.userService.UpdateUserRole(r.Context(), targetID, req.Role)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(w, "User not found")
			return
		}
		response.InternalServerError(w, "Failed to update user role", err.Error())
		return
	}

	response.OK(w, "User role updated successfully", user)
}
