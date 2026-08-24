package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gozaq/internal/domain"
	"gozaq/internal/handler"
	"gozaq/pkg/pagination"
)

// MockUserService is a mock implementation of domain.UserService
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) Register(ctx context.Context, req domain.RegisterRequest) (*domain.AuthResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AuthResponse), args.Error(1)
}

func (m *MockUserService) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AuthResponse), args.Error(1)
}

func (m *MockUserService) RefreshToken(ctx context.Context, req domain.RefreshTokenRequest) (*domain.AuthResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AuthResponse), args.Error(1)
}

func (m *MockUserService) Logout(ctx context.Context, userID uuid.UUID, refreshToken string) error {
	args := m.Called(ctx, userID, refreshToken)
	return args.Error(0)
}

func (m *MockUserService) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.UserResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserResponse), args.Error(1)
}

func (m *MockUserService) UpdateProfile(ctx context.Context, userID uuid.UUID, req domain.UpdateProfileRequest) (*domain.UserResponse, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserResponse), args.Error(1)
}

func (m *MockUserService) UpdateUserRole(ctx context.Context, userID uuid.UUID, role string) (*domain.UserResponse, error) {
	args := m.Called(ctx, userID, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserResponse), args.Error(1)
}

func (m *MockUserService) ListUsers(ctx context.Context, p pagination.Params, search string) (*domain.PaginatedUsersResponse, error) {
	args := m.Called(ctx, p, search)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PaginatedUsersResponse), args.Error(1)
}

func (m *MockUserService) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func TestUserHandler_Register_Success(t *testing.T) {
	mockService := new(MockUserService)
	userHandler := handler.NewUserHandler(mockService)

	reqBody := domain.RegisterRequest{
		Name:     "Zaq",
		Email:    "zaq@example.com",
		Password: "password123",
	}

	authResp := &domain.AuthResponse{
		AccessToken:  "mock-access-token",
		RefreshToken: "mock-refresh-token",
		User: domain.UserResponse{
			ID:        uuid.New(),
			Name:      "Zaq",
			Email:     "zaq@example.com",
			Role:      "user",
			CreatedAt: time.Now(),
		},
	}

	mockService.On("Register", mock.Anything, reqBody).Return(authResp, nil)

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	userHandler.Register(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp["success"].(bool))
	assert.Equal(t, "User registered successfully", resp["message"])
	mockService.AssertExpectations(t)
}

func TestUserHandler_Register_ValidationError(t *testing.T) {
	mockService := new(MockUserService)
	userHandler := handler.NewUserHandler(mockService)

	// Missing required fields
	reqBody := domain.RegisterRequest{
		Name:     "",
		Email:    "invalid-email",
		Password: "123",
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	userHandler.Register(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	var resp map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.False(t, resp["success"].(bool))
	assert.Equal(t, "VALIDATION_ERROR", resp["error_code"])
}

func TestUserHandler_Login_Success(t *testing.T) {
	mockService := new(MockUserService)
	userHandler := handler.NewUserHandler(mockService)

	reqBody := domain.LoginRequest{
		Email:    "zaq@example.com",
		Password: "password123",
	}

	authResp := &domain.AuthResponse{
		AccessToken:  "mock-access-token",
		RefreshToken: "mock-refresh-token",
		User: domain.UserResponse{
			ID:    uuid.New(),
			Email: "zaq@example.com",
		},
	}

	mockService.On("Login", mock.Anything, reqBody).Return(authResp, nil)

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	userHandler.Login(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp["success"].(bool))
	mockService.AssertExpectations(t)
}
