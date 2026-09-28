package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	"gozaq/pkg/pagination"
)

// User entity represents user data in core business domain
type User struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Password    string     `json:"-"`
	Role        string     `json:"role"`
	IDNumber    string     `json:"id_number,omitempty"`
	Affiliation string     `json:"affiliation,omitempty"`
	IsVerified  bool       `json:"is_verified"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Request & Response DTOs
type RegisterRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=6,max=50"`
	Role        string `json:"role,omitempty" validate:"omitempty,oneof=user manager admin mahasiswa dosen umum"`
	IDNumber    string `json:"id_number,omitempty"`
	Affiliation string `json:"affiliation,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

type ResendVerificationRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type UpdateProfileRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Affiliation string `json:"affiliation,omitempty"`
}

type AdminUpdateUserRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Email       string `json:"email" validate:"required,email"`
	Role        string `json:"role" validate:"required,oneof=user manager admin mahasiswa dosen umum"`
	IDNumber    string `json:"id_number,omitempty"`
	Affiliation string `json:"affiliation,omitempty"`
	IsVerified  *bool  `json:"is_verified,omitempty"`
}

type UpdateRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=user manager admin mahasiswa dosen umum"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6,max=50"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6,max=50"`
}

type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         UserResponse `json:"user"`
}

type UserResponse struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	IDNumber    string     `json:"id_number,omitempty"`
	Affiliation string     `json:"affiliation,omitempty"`
	IsVerified  bool       `json:"is_verified"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
	Permissions []string   `json:"permissions"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type PaginatedUsersResponse struct {
	Users []UserResponse  `json:"users"`
	Meta  pagination.Meta `json:"meta"`
}

func (u *User) ToResponse(permissions ...string) UserResponse {
	if permissions == nil {
		permissions = []string{}
	}
	return UserResponse{
		ID:          u.ID,
		Name:        u.Name,
		Email:       u.Email,
		Role:        u.Role,
		IDNumber:    u.IDNumber,
		Affiliation: u.Affiliation,
		IsVerified:  u.IsVerified,
		VerifiedAt:  u.VerifiedAt,
		Permissions: permissions,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

// Interfaces / Ports
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	UpdatePassword(ctx context.Context, id uuid.UUID, hashedPassword string) error
	UpdateRole(ctx context.Context, id uuid.UUID, role string) error
	SetEmailVerified(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, p pagination.Params, search string) ([]*User, int, error)
}

type UserService interface {
	Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	RefreshToken(ctx context.Context, req RefreshTokenRequest) (*AuthResponse, error)
	VerifyEmail(ctx context.Context, req VerifyEmailRequest) error
	ResendVerification(ctx context.Context, req ResendVerificationRequest) error
	ForgotPassword(ctx context.Context, req ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req ResetPasswordRequest) error
	Logout(ctx context.Context, userID uuid.UUID, refreshToken string) error
	GetProfile(ctx context.Context, userID uuid.UUID) (*UserResponse, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (*UserResponse, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, req ChangePasswordRequest) error
	UpdateUserRole(ctx context.Context, userID uuid.UUID, role string) (*UserResponse, error)
	UpdateUser(ctx context.Context, userID uuid.UUID, req AdminUpdateUserRequest) (*UserResponse, error)
	ListUsers(ctx context.Context, p pagination.Params, search string) (*PaginatedUsersResponse, error)
	DeleteUser(ctx context.Context, userID uuid.UUID) error
}
