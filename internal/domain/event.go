package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID                   string          `json:"id"`
	Title                string          `json:"title"`
	Slug                 string          `json:"slug"`
	Description          string          `json:"description"`
	LongDescription      string          `json:"long_description"`
	SeriesID             *string         `json:"series_id"`
	SeriesName           *string         `json:"series_name"`
	CategoryName         string          `json:"category_name"`
	CategoryVariant      string          `json:"category_variant"`
	EventDate            string          `json:"event_date"` // YYYY-MM-DD
	DateDay              string          `json:"date_day"`
	DateMonth            string          `json:"date_month"`
	DateYear             string          `json:"date_year"`
	DateFullText         string          `json:"date_full_text"`
	Time                 string          `json:"time"`
	LocationName         string          `json:"location_name"`
	LocationRoom         *string         `json:"location_room"`
	LocationAddress      *string         `json:"location_address"`
	LocationType         string          `json:"location_type"` // offline, online, hybrid
	CoverImage           string          `json:"cover_image"`
	QuotaTotal           int             `json:"quota_total"`
	QuotaFilled          int             `json:"quota_filled"`
	QuotaStatus          string          `json:"quota_status"`       // open, closing-soon, full
	QuotaStatusLabel     string          `json:"quota_status_label"` // Pendaftaran Dibuka, Kuota Penuh, dll
	RegistrationURL      *string         `json:"registration_url"`
	Featured             bool            `json:"featured"`
	IsFree               bool            `json:"is_free"`
	Price                int64           `json:"price"`
	PaymentInfo          json.RawMessage `json:"payment_info"`
	RequiresApproval     bool            `json:"requires_approval"`
	CustomFieldsSchema   json.RawMessage `json:"custom_fields_schema"`
	Speakers             json.RawMessage `json:"speakers"`
	Rundown              json.RawMessage `json:"rundown"`
	Benefits             json.RawMessage `json:"benefits"`
	Prerequisites        json.RawMessage `json:"prerequisites"`
	TargetAudience       *string         `json:"target_audience"`
	RegistrationDeadline *string         `json:"registration_deadline"`
	Fee                  string          `json:"fee"`
	ContactPerson        json.RawMessage `json:"contact_person"`
	Status               string          `json:"status"` // draft, published, cancelled, completed
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

type EventCustomField struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Type        string   `json:"type"` // text, number, textarea, select, file
	Required    bool     `json:"required"`
	Placeholder string   `json:"placeholder,omitempty"`
	Options     []string `json:"options,omitempty"`
	Accept      string   `json:"accept,omitempty"`
}

type EventPaymentInfo struct {
	BankName      string `json:"bank_name"`
	AccountNumber string `json:"account_number"`
	AccountHolder string `json:"account_holder"`
	QRISImageURL  string `json:"qris_image_url"`
	Instructions  string `json:"instructions"`
}

type EventRegistration struct {
	ID               string          `json:"id"`
	RegistrationCode string          `json:"registration_code"`
	EventID          string          `json:"event_id"`
	EventTitle       string          `json:"event_title,omitempty"`
	EventSlug        string          `json:"event_slug,omitempty"`
	EventDate        string          `json:"event_date,omitempty"`
	EventTime        string          `json:"event_time,omitempty"`
	EventLocation    string          `json:"event_location,omitempty"`
	UserID           uuid.UUID       `json:"user_id"`
	FullName         string          `json:"full_name"`
	IdentityNumber   *string         `json:"identity_number"`
	Institution      string          `json:"institution"`
	Email            string          `json:"email"`
	Phone            string          `json:"phone"`
	Notes            *string         `json:"notes"`
	Answers          json.RawMessage `json:"answers"`
	UploadedFiles    json.RawMessage `json:"uploaded_files"`
	PaymentProofURL  *string         `json:"payment_proof_url"`
	Status           string          `json:"status"` // pending_review, needs_revision, approved, rejected, attended, cancelled
	AdminNotes       *string         `json:"admin_notes"`
	VerifiedBy       *uuid.UUID      `json:"verified_by"`
	VerifiedAt       *time.Time      `json:"verified_at"`
	AttendedAt       *time.Time      `json:"attended_at"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type CreateEventRequest struct {
	Title                string              `json:"title" validate:"required"`
	Slug                 string              `json:"slug"`
	Description          string              `json:"description" validate:"required"`
	LongDescription      string              `json:"long_description"`
	SeriesID             *string             `json:"series_id"`
	SeriesName           *string             `json:"series_name"`
	CategoryName         string              `json:"category_name" validate:"required"`
	CategoryVariant      string              `json:"category_variant"`
	EventDate            string              `json:"event_date" validate:"required"` // YYYY-MM-DD
	DateDay              string              `json:"date_day"`
	DateMonth            string              `json:"date_month"`
	DateYear             string              `json:"date_year"`
	DateFullText         string              `json:"date_full_text"`
	Time                 string              `json:"time" validate:"required"`
	LocationName         string              `json:"location_name" validate:"required"`
	LocationRoom         *string             `json:"location_room"`
	LocationAddress      *string             `json:"location_address"`
	LocationType         string              `json:"location_type"`
	CoverImage           string              `json:"cover_image" validate:"required"`
	QuotaTotal           int                 `json:"quota_total" validate:"required,min=1"`
	Featured             bool                `json:"featured"`
	IsFree               bool                `json:"is_free"`
	Price                int64               `json:"price"`
	PaymentInfo          *EventPaymentInfo   `json:"payment_info"`
	RequiresApproval     bool                `json:"requires_approval"`
	CustomFieldsSchema   []EventCustomField  `json:"custom_fields_schema"`
	Speakers             []interface{}       `json:"speakers"`
	Rundown              []interface{}       `json:"rundown"`
	Benefits             []string            `json:"benefits"`
	Prerequisites        []string            `json:"prerequisites"`
	TargetAudience       *string             `json:"target_audience"`
	RegistrationDeadline *string             `json:"registration_deadline"`
	Fee                  string              `json:"fee"`
	ContactPerson        interface{}         `json:"contact_person"`
	Status               string              `json:"status"`
}

type UpdateEventRequest struct {
	Title                *string             `json:"title"`
	Slug                 *string             `json:"slug"`
	Description          *string             `json:"description"`
	LongDescription      *string             `json:"long_description"`
	SeriesID             *string             `json:"series_id"`
	SeriesName           *string             `json:"series_name"`
	CategoryName         *string             `json:"category_name"`
	CategoryVariant      *string             `json:"category_variant"`
	EventDate            *string             `json:"event_date"`
	DateDay              *string             `json:"date_day"`
	DateMonth            *string             `json:"date_month"`
	DateYear             *string             `json:"date_year"`
	DateFullText         *string             `json:"date_full_text"`
	Time                 *string             `json:"time"`
	LocationName         *string             `json:"location_name"`
	LocationRoom         *string             `json:"location_room"`
	LocationAddress      *string             `json:"location_address"`
	LocationType         *string             `json:"location_type"`
	CoverImage           *string             `json:"cover_image"`
	QuotaTotal           *int                `json:"quota_total"`
	QuotaStatus          *string             `json:"quota_status"`
	QuotaStatusLabel     *string             `json:"quota_status_label"`
	Featured             *bool               `json:"featured"`
	IsFree               *bool               `json:"is_free"`
	Price                *int64              `json:"price"`
	PaymentInfo          *EventPaymentInfo   `json:"payment_info"`
	RequiresApproval     *bool               `json:"requires_approval"`
	CustomFieldsSchema   *[]EventCustomField `json:"custom_fields_schema"`
	Speakers             *[]interface{}      `json:"speakers"`
	Rundown              *[]interface{}      `json:"rundown"`
	Benefits             *[]string           `json:"benefits"`
	Prerequisites        *[]string           `json:"prerequisites"`
	TargetAudience       *string             `json:"target_audience"`
	RegistrationDeadline *string             `json:"registration_deadline"`
	Fee                  *string             `json:"fee"`
	ContactPerson        interface{}         `json:"contact_person"`
	Status               *string             `json:"status"`
}

type RegisterEventRequest struct {
	FullName        string                 `json:"full_name" validate:"required"`
	IdentityNumber  string                 `json:"identity_number"`
	Institution     string                 `json:"institution" validate:"required"`
	Email           string                 `json:"email" validate:"required,email"`
	Phone           string                 `json:"phone" validate:"required"`
	Notes           string                 `json:"notes"`
	Answers         map[string]interface{} `json:"answers"`
	UploadedFiles   []string               `json:"uploaded_files"`
	PaymentProofURL *string                `json:"payment_proof_url"`
}

type UpdateRegistrationStatusRequest struct {
	Status     string  `json:"status" validate:"required,oneof=pending_review needs_revision approved rejected attended cancelled"`
	AdminNotes *string `json:"admin_notes"`
}

type ReviseRegistrationRequest struct {
	Notes           *string                `json:"notes"`
	Answers         map[string]interface{} `json:"answers"`
	UploadedFiles   []string               `json:"uploaded_files"`
	PaymentProofURL *string                `json:"payment_proof_url"`
}

type EventCheckInRequest struct {
	RegistrationCode string `json:"registration_code" validate:"required"`
}

type EventFilter struct {
	Category     string
	SeriesID     string
	Status       string
	Search       string
	UpcomingOnly bool
	PastOnly     bool
	Page         int
	Limit        int
}

type RegistrationFilter struct {
	EventID string
	UserID  *uuid.UUID
	Status  string
	Search  string
	Page    int
	Limit   int
}

type EventPagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

type EventListResponse struct {
	Events     []Event         `json:"events"`
	Pagination EventPagination `json:"pagination"`
}

type RegistrationListResponse struct {
	Registrations []EventRegistration `json:"registrations"`
	Pagination    EventPagination     `json:"pagination"`
}

type EventRepository interface {
	Create(ctx context.Context, e *Event) error
	GetByID(ctx context.Context, id string) (*Event, error)
	GetBySlug(ctx context.Context, slug string) (*Event, error)
	List(ctx context.Context, filter EventFilter) ([]Event, int, error)
	Update(ctx context.Context, id string, req *UpdateEventRequest) (*Event, error)
	Delete(ctx context.Context, id string) error
	IncrementQuotaFilled(ctx context.Context, id string, delta int) error

	Register(ctx context.Context, reg *EventRegistration) error
	GetRegistrationByID(ctx context.Context, id string) (*EventRegistration, error)
	GetRegistrationByCode(ctx context.Context, code string) (*EventRegistration, error)
	GetUserRegistration(ctx context.Context, eventID string, userID uuid.UUID) (*EventRegistration, error)
	ListRegistrations(ctx context.Context, filter RegistrationFilter) ([]EventRegistration, int, error)
	UpdateRegistrationStatus(ctx context.Context, id string, status string, notes *string, verifiedBy *uuid.UUID) error
	ReviseRegistration(ctx context.Context, id string, req *ReviseRegistrationRequest) error
	CheckIn(ctx context.Context, code string) (*EventRegistration, error)
}
