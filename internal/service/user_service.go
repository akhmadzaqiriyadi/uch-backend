package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/singleflight"

	"gozaq/config"
	"gozaq/internal/domain"
	"gozaq/pkg/cache"
	"gozaq/pkg/mailer"
	"gozaq/pkg/pagination"
	"gozaq/pkg/worker"
)

type UserService struct {
	repo     domain.UserRepository
	rbacRepo domain.RBACRepository
	cache    cache.Cache
	worker   *worker.Pool
	mailer   mailer.Mailer
	cfg      *config.Config
	sfGroup  singleflight.Group
}

type JWTClaims struct {
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	Permissions []string  `json:"permissions,omitempty"`
	jwt.RegisteredClaims
}

func NewUserService(
	repo domain.UserRepository,
	rbacRepo domain.RBACRepository,
	cache cache.Cache,
	workerPool *worker.Pool,
	mailer mailer.Mailer,
	cfg *config.Config,
) *UserService {
	return &UserService{
		repo:     repo,
		rbacRepo: rbacRepo,
		cache:    cache,
		worker:   workerPool,
		mailer:   mailer,
		cfg:      cfg,
	}
}

func (s *UserService) Register(ctx context.Context, req domain.RegisterRequest) (*domain.AuthResponse, error) {
	existingUser, err := s.repo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, domain.ErrAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	role := "user"
	if req.Role != "" {
		role = req.Role
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID:        uuid.New(),
		Name:      req.Name,
		Email:     req.Email,
		Password:  string(hashedPassword),
		Role:      role,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	permissions := s.getRolePermissions(ctx, user.Role)

	accessToken, refreshToken, err := s.issueTokens(ctx, user, permissions)
	if err != nil {
		return nil, err
	}

	userEmail := user.Email
	userName := user.Name
	s.worker.Submit(func(ctx context.Context) error {
		return s.mailer.SendWelcomeEmail(ctx, userEmail, userName)
	})

	return &domain.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user.ToResponse(permissions...),
	}, nil
}

func (s *UserService) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) {
	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	permissions := s.getRolePermissions(ctx, user.Role)

	accessToken, refreshToken, err := s.issueTokens(ctx, user, permissions)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user.ToResponse(permissions...),
	}, nil
}

func (s *UserService) RefreshToken(ctx context.Context, req domain.RefreshTokenRequest) (*domain.AuthResponse, error) {
	// 1. Look up refresh token in Redis
	redisKey := fmt.Sprintf("refresh_token:%s", req.RefreshToken)
	var userIDStr string
	if err := s.cache.Get(ctx, redisKey, &userIDStr); err != nil {
		return nil, domain.ErrUnauthorized
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	// 2. Token Rotation: Revoke old refresh token from Redis
	_ = s.cache.Delete(ctx, redisKey)

	permissions := s.getRolePermissions(ctx, user.Role)

	// 3. Issue brand new Access Token & Refresh Token
	accessToken, refreshToken, err := s.issueTokens(ctx, user, permissions)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user.ToResponse(permissions...),
	}, nil
}

func (s *UserService) Logout(ctx context.Context, userID uuid.UUID, refreshToken string) error {
	// Revoke refresh token
	if refreshToken != "" {
		_ = s.cache.Delete(ctx, fmt.Sprintf("refresh_token:%s", refreshToken))
	}
	// Evict user profile cache
	_ = s.cache.Delete(ctx, fmt.Sprintf("user:profile:%s", userID.String()))
	return nil
}

func (s *UserService) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.UserResponse, error) {
	cacheKey := fmt.Sprintf("user:profile:%s", userID.String())

	var cachedProfile domain.UserResponse
	if err := s.cache.Get(ctx, cacheKey, &cachedProfile); err == nil {
		slog.Debug("Redis cache hit for user profile", slog.String("user_id", userID.String()))
		return &cachedProfile, nil
	}

	// SingleFlight ensures only 1 concurrent DB query per cacheKey (prevents Cache Stampede)
	v, err, _ := s.sfGroup.Do(cacheKey, func() (any, error) {
		user, err := s.repo.GetByID(ctx, userID)
		if err != nil {
			return nil, err
		}

		permissions := s.getRolePermissions(ctx, user.Role)
		resp := user.ToResponse(permissions...)
		_ = s.cache.Set(ctx, cacheKey, resp, 15*time.Minute)
		return &resp, nil
	})

	if err != nil {
		return nil, err
	}

	return v.(*domain.UserResponse), nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID uuid.UUID, req domain.UpdateProfileRequest) (*domain.UserResponse, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	user.Name = req.Name
	user.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	// Invalidate Redis profile cache
	_ = s.cache.Delete(ctx, fmt.Sprintf("user:profile:%s", userID.String()))

	permissions := s.getRolePermissions(ctx, user.Role)
	resp := user.ToResponse(permissions...)
	return &resp, nil
}

func (s *UserService) UpdateUserRole(ctx context.Context, userID uuid.UUID, role string) (*domain.UserResponse, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateRole(ctx, userID, role); err != nil {
		return nil, err
	}

	user.Role = role
	user.UpdatedAt = time.Now().UTC()

	// Invalidate Redis profile cache
	_ = s.cache.Delete(ctx, fmt.Sprintf("user:profile:%s", userID.String()))

	permissions := s.getRolePermissions(ctx, role)
	resp := user.ToResponse(permissions...)
	return &resp, nil
}

func (s *UserService) ListUsers(ctx context.Context, p pagination.Params, search string) (*domain.PaginatedUsersResponse, error) {
	users, totalItems, err := s.repo.List(ctx, p, search)
	if err != nil {
		return nil, err
	}

	userResponses := make([]domain.UserResponse, len(users))
	for i, u := range users {
		userResponses[i] = u.ToResponse()
	}

	meta := pagination.BuildMeta(p, totalItems)

	return &domain.PaginatedUsersResponse{
		Users: userResponses,
		Meta:  meta,
	}, nil
}

func (s *UserService) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	if err := s.repo.Delete(ctx, userID); err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.Delete(ctx, fmt.Sprintf("user:profile:%s", userID.String()))
	return nil
}

func (s *UserService) getRolePermissions(ctx context.Context, roleID string) []string {
	if s.rbacRepo == nil {
		return []string{}
	}

	cacheKey := fmt.Sprintf("role:permissions:%s", roleID)
	var perms []string
	if err := s.cache.Get(ctx, cacheKey, &perms); err == nil && len(perms) > 0 {
		return perms
	}

	perms, err := s.rbacRepo.GetPermissionsByRole(ctx, roleID)
	if err != nil {
		slog.Warn("Failed to fetch role permissions from database", slog.String("role", roleID), slog.String("error", err.Error()))
		return []string{}
	}

	_ = s.cache.Set(ctx, cacheKey, perms, 1*time.Hour)
	return perms
}

func (s *UserService) issueTokens(ctx context.Context, user *domain.User, permissions []string) (string, string, error) {
	// 1. Generate Access Token (15 Minutes) with Role and Granular Permissions
	claims := JWTClaims{
		UserID:      user.ID,
		Email:       user.Email,
		Role:        user.Role,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// 2. Generate Refresh Token (Random 32-byte hex string)
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	refreshToken := hex.EncodeToString(randomBytes)

	// 3. Store Refresh Token in Redis with 7 days TTL
	redisKey := fmt.Sprintf("refresh_token:%s", refreshToken)
	if err := s.cache.Set(ctx, redisKey, user.ID.String(), 7*24*time.Hour); err != nil {
		slog.Warn("Failed to cache refresh token in Redis", slog.String("error", err.Error()))
	}

	return accessToken, refreshToken, nil
}
