package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"

	"gozaq/config"
	"gozaq/internal/domain"
	"gozaq/internal/service"
	"gozaq/pkg/cache"
	"gozaq/pkg/mailer"
	"gozaq/pkg/pagination"
	"gozaq/pkg/worker"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, hashedPassword string) error {
	args := m.Called(ctx, id, hashedPassword)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateRole(ctx context.Context, id uuid.UUID, role string) error {
	args := m.Called(ctx, id, role)
	return args.Error(0)
}

func (m *MockUserRepository) SetEmailVerified(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, p pagination.Params, search string) ([]*domain.User, int, error) {
	args := m.Called(ctx, p, search)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*domain.User), args.Int(1), args.Error(2)
}

type MockRBACRepository struct {
	mock.Mock
}

func (m *MockRBACRepository) GetPermissionsByRole(ctx context.Context, roleID string) ([]string, error) {
	return []string{"users:read", "uploads:create"}, nil
}

func (m *MockRBACRepository) ListRoles(ctx context.Context) ([]domain.RoleWithPermissions, error) {
	return nil, nil
}

func (m *MockRBACRepository) ListPermissions(ctx context.Context) ([]domain.Permission, error) {
	return nil, nil
}

func (m *MockRBACRepository) AssignPermissionToRole(ctx context.Context, roleID, permissionID string) error {
	return nil
}

func (m *MockRBACRepository) RevokePermissionFromRole(ctx context.Context, roleID, permissionID string) error {
	return nil
}

func (m *MockRBACRepository) CreateRole(ctx context.Context, id, name, description string) error {
	return nil
}

func (m *MockRBACRepository) DeleteRole(ctx context.Context, id string) error {
	return nil
}

type MockAuditRepository struct {
	mock.Mock
}

func (m *MockAuditRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	return nil
}

func (m *MockAuditRepository) List(ctx context.Context, p pagination.Params, action string) ([]domain.AuditLog, int, error) {
	return nil, 0, nil
}

func setupTest() (*MockUserRepository, *service.UserService) {
	mockRepo := new(MockUserRepository)
	mockRBAC := new(MockRBACRepository)
	mockAudit := new(MockAuditRepository)
	mockCache := cache.NewNoopCache()
	mockMailer := mailer.NewLogMailer()
	workerPool := worker.NewPool(1, 10)
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:        "test-secret-key-1234567890",
			ExpireMinutes: 60,
		},
	}
	userService := service.NewUserService(mockRepo, mockRBAC, mockAudit, mockCache, workerPool, mockMailer, cfg)
	return mockRepo, userService
}

func TestUserService_Register_Success(t *testing.T) {
	mockRepo, userService := setupTest()
	ctx := context.Background()

	req := domain.RegisterRequest{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "password123",
	}

	mockRepo.On("GetByEmail", ctx, req.Email).Return(nil, domain.ErrNotFound)
	mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.User")).Return(nil)

	resp, err := userService.Register(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, req.Name, resp.User.Name)
	assert.Equal(t, req.Email, resp.User.Email)
	assert.Equal(t, "user", resp.User.Role)
	mockRepo.AssertExpectations(t)
}

func TestUserService_Login_Success(t *testing.T) {
	mockRepo, userService := setupTest()
	ctx := context.Background()

	rawPassword := "mypassword123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)

	user := &domain.User{
		ID:        uuid.New(),
		Name:      "Login User",
		Email:     "login@example.com",
		Password:  string(hashedPassword),
		Role:      "user",
		CreatedAt: time.Now(),
	}

	mockRepo.On("GetByEmail", ctx, user.Email).Return(user, nil)

	resp, err := userService.Login(ctx, domain.LoginRequest{
		Email:    user.Email,
		Password: rawPassword,
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, user.Email, resp.User.Email)
	mockRepo.AssertExpectations(t)
}

func TestUserService_ForgotPassword(t *testing.T) {
	mockRepo, userService := setupTest()
	ctx := context.Background()

	user := &domain.User{
		ID:        uuid.New(),
		Name:      "Forgot User",
		Email:     "forgot@example.com",
		Role:      "user",
		CreatedAt: time.Now(),
	}

	mockRepo.On("GetByEmail", ctx, "forgot@example.com").Return(user, nil)

	err := userService.ForgotPassword(ctx, domain.ForgotPasswordRequest{
		Email: "forgot@example.com",
	})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUserService_ResendVerification(t *testing.T) {
	mockRepo, userService := setupTest()
	ctx := context.Background()

	user := &domain.User{
		ID:         uuid.New(),
		Name:       "Verify User",
		Email:      "verify@example.com",
		Role:       "user",
		IsVerified: false,
		CreatedAt:  time.Now(),
	}

	mockRepo.On("GetByEmail", ctx, "verify@example.com").Return(user, nil)

	err := userService.ResendVerification(ctx, domain.ResendVerificationRequest{
		Email: "verify@example.com",
	})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
