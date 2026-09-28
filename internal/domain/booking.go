package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Booking struct {
	ID            string    `json:"id"`
	UserID        uuid.UUID `json:"user_id"`
	RoomID        string    `json:"room_id"`
	RoomName      string    `json:"room_name"`
	ApplicantName string    `json:"applicant_name"`
	ApplicantRole string    `json:"applicant_role"`
	IDNumber      string    `json:"id_number"`
	Prodi         string    `json:"prodi"`
	Purpose       string    `json:"purpose"`
	Audience      int       `json:"audience"`
	BookingDate   string    `json:"booking_date"` // YYYY-MM-DD
	StartTime     string    `json:"start_time"`   // HH:MM
	EndTime       string    `json:"end_time"`     // HH:MM
	Status        string    `json:"status"`       // pending, approved, rejected, cancelled, completed
	AdminNotes    *string   `json:"admin_notes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateBookingRequest struct {
	RoomID        string `json:"room_id" validate:"required"`
	RoomName      string `json:"room_name"`
	ApplicantName string `json:"applicant_name" validate:"required"`
	ApplicantRole string `json:"applicant_role" validate:"required"`
	IDNumber      string `json:"id_number"`
	Prodi         string `json:"prodi"`
	Purpose       string `json:"purpose" validate:"required"`
	Audience      int    `json:"audience" validate:"required,min=1"`
	BookingDate   string `json:"booking_date" validate:"required"`
	StartTime     string `json:"start_time" validate:"required"`
	EndTime       string `json:"end_time" validate:"required"`
}

type UpdateBookingStatusRequest struct {
	Status string  `json:"status" validate:"required,oneof=pending approved rejected cancelled completed"`
	Notes  *string `json:"notes"`
}

type AdminCheckInRequest struct {
	BookingCode string `json:"booking_code" validate:"required"`
}

type SelfCheckInRequest struct {
	RoomID      string `json:"room_id"`
	BookingCode string `json:"booking_code"`
}

type BookingFilter struct {
	UserID *uuid.UUID
	RoomID string
	Status string
	Search string
	Page   int
	Limit  int
}

type BookingPagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

type BookingListResponse struct {
	Bookings   []Booking         `json:"bookings"`
	Pagination BookingPagination `json:"pagination"`
}

type BookingRepository interface {
	Create(ctx context.Context, b *Booking) error
	GetByID(ctx context.Context, id string) (*Booking, error)
	List(ctx context.Context, filter BookingFilter) ([]Booking, int, error)
	UpdateStatus(ctx context.Context, id string, status string, notes *string) error
	Delete(ctx context.Context, id string) error
}

