package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"gozaq/internal/domain"
	"gozaq/internal/service"
)

type mockEventRepo struct {
	events        map[string]*domain.Event
	registrations map[string]*domain.EventRegistration
}

func newMockEventRepo() *mockEventRepo {
	return &mockEventRepo{
		events:        make(map[string]*domain.Event),
		registrations: make(map[string]*domain.EventRegistration),
	}
}

func (m *mockEventRepo) Create(ctx context.Context, e *domain.Event) error {
	m.events[e.ID] = e
	m.events[e.Slug] = e
	return nil
}

func (m *mockEventRepo) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	e, ok := m.events[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return e, nil
}

func (m *mockEventRepo) GetBySlug(ctx context.Context, slug string) (*domain.Event, error) {
	e, ok := m.events[slug]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return e, nil
}

func (m *mockEventRepo) List(ctx context.Context, filter domain.EventFilter) ([]domain.Event, int, error) {
	list := make([]domain.Event, 0)
	for _, e := range m.events {
		list = append(list, *e)
	}
	return list, len(list), nil
}

func (m *mockEventRepo) Update(ctx context.Context, id string, req *domain.UpdateEventRequest) (*domain.Event, error) {
	e, ok := m.events[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return e, nil
}

func (m *mockEventRepo) Delete(ctx context.Context, id string) error {
	delete(m.events, id)
	return nil
}

func (m *mockEventRepo) IncrementQuotaFilled(ctx context.Context, id string, delta int) error {
	e, ok := m.events[id]
	if ok {
		e.QuotaFilled += delta
	}
	return nil
}

func (m *mockEventRepo) Register(ctx context.Context, reg *domain.EventRegistration) error {
	m.registrations[reg.ID] = reg
	return nil
}

func (m *mockEventRepo) GetRegistrationByID(ctx context.Context, id string) (*domain.EventRegistration, error) {
	reg, ok := m.registrations[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return reg, nil
}

func (m *mockEventRepo) GetRegistrationByCode(ctx context.Context, code string) (*domain.EventRegistration, error) {
	for _, reg := range m.registrations {
		if reg.RegistrationCode == code {
			return reg, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockEventRepo) GetUserRegistration(ctx context.Context, eventID string, userID uuid.UUID) (*domain.EventRegistration, error) {
	for _, reg := range m.registrations {
		if reg.EventID == eventID && reg.UserID == userID {
			return reg, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockEventRepo) ListRegistrations(ctx context.Context, filter domain.RegistrationFilter) ([]domain.EventRegistration, int, error) {
	list := make([]domain.EventRegistration, 0)
	for _, reg := range m.registrations {
		list = append(list, *reg)
	}
	return list, len(list), nil
}

func (m *mockEventRepo) UpdateRegistrationStatus(ctx context.Context, id string, status string, notes *string, verifiedBy *uuid.UUID) error {
	reg, ok := m.registrations[id]
	if !ok {
		return domain.ErrNotFound
	}
	reg.Status = status
	reg.AdminNotes = notes
	return nil
}

func (m *mockEventRepo) ReviseRegistration(ctx context.Context, id string, req *domain.ReviseRegistrationRequest) error {
	reg, ok := m.registrations[id]
	if !ok {
		return domain.ErrNotFound
	}
	reg.Status = "pending_review"
	return nil
}

func (m *mockEventRepo) CheckIn(ctx context.Context, code string) (*domain.EventRegistration, error) {
	for _, reg := range m.registrations {
		if reg.RegistrationCode == code {
			reg.Status = "attended"
			now := time.Now()
			reg.AttendedAt = &now
			return reg, nil
		}
	}
	return nil, domain.ErrNotFound
}

func TestEventService_Register_AutoApproveForFreeEvent(t *testing.T) {
	repo := newMockEventRepo()
	svc := service.NewEventService(repo, nil)

	// Create future free event
	futureDate := time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	event := &domain.Event{
		ID:               "evt-free-1",
		Title:            "Workshop Gratis IoT",
		Slug:             "workshop-gratis-iot",
		EventDate:        futureDate,
		Status:           "published",
		QuotaTotal:       30,
		QuotaFilled:      5,
		IsFree:           true,
		RequiresApproval: false,
	}
	_ = repo.Create(context.Background(), event)

	userID := uuid.New()
	regReq := &domain.RegisterEventRequest{
		FullName:    "Mahasiswa UTY",
		Institution: "UTY",
		Email:       "mhs@uty.ac.id",
		Phone:       "081234567890",
	}

	reg, err := svc.Register(context.Background(), userID, event.ID, regReq)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if reg.Status != "approved" {
		t.Errorf("expected auto-approved status for free event, got %s", reg.Status)
	}

	if reg.RegistrationCode == "" {
		t.Errorf("expected non-empty ticket code")
	}

	// Double registration check
	_, err = svc.Register(context.Background(), userID, event.ID, regReq)
	if err == nil {
		t.Fatalf("expected error for duplicate registration, got nil")
	}
}

func TestEventService_Register_PendingReviewForPaidEvent(t *testing.T) {
	repo := newMockEventRepo()
	svc := service.NewEventService(repo, nil)

	futureDate := time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	event := &domain.Event{
		ID:               "evt-paid-1",
		Title:            "Bootcamp AI Intensif",
		Slug:             "bootcamp-ai-intensif",
		EventDate:        futureDate,
		Status:           "published",
		QuotaTotal:       20,
		QuotaFilled:      0,
		IsFree:           false,
		Price:            50000,
		RequiresApproval: true,
	}
	_ = repo.Create(context.Background(), event)

	userID := uuid.New()
	regReq := &domain.RegisterEventRequest{
		FullName:    "Peserta Bootcamp",
		Institution: "UTY",
		Email:       "bootcamp@uty.ac.id",
		Phone:       "081234567890",
	}

	reg, err := svc.Register(context.Background(), userID, event.ID, regReq)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if reg.Status != "pending_review" {
		t.Errorf("expected pending_review status for paid event, got %s", reg.Status)
	}
}
