package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"

	"gozaq/internal/domain"
)

type EventService struct {
	eventRepo domain.EventRepository
	auditRepo domain.AuditRepository
	notifier  RealtimeNotifier
}

func NewEventService(
	eventRepo domain.EventRepository,
	auditRepo domain.AuditRepository,
) *EventService {
	return &EventService{
		eventRepo: eventRepo,
		auditRepo: auditRepo,
	}
}

func (s *EventService) SetNotifier(notifier RealtimeNotifier) {
	s.notifier = notifier
}

func (s *EventService) logAudit(ctx context.Context, userID *uuid.UUID, action, entityID string, details any) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Create(ctx, &domain.AuditLog{
		ID:        uuid.New(),
		UserID:    userID,
		Action:    action,
		Entity:    "events",
		EntityID:  entityID,
		Details:   details,
		CreatedAt: time.Now().UTC(),
	})
}

func generateRandomCode(prefix string) string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 5)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return fmt.Sprintf("%s-%s-%d", prefix, string(b), time.Now().Year())
}

func slugify(text string) string {
	lower := strings.ToLower(strings.TrimSpace(text))
	var sb strings.Builder
	for _, r := range lower {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			sb.WriteRune('-')
		}
	}
	slug := sb.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	return strings.Trim(slug, "-")
}

func (s *EventService) CreateEvent(ctx context.Context, actorID uuid.UUID, req *domain.CreateEventRequest) (*domain.Event, error) {
	slug := req.Slug
	if slug == "" {
		slug = slugify(req.Title)
	}

	parsedDate, err := time.Parse("2006-01-02", req.EventDate)
	if err != nil {
		return nil, fmt.Errorf("invalid event_date format (must be YYYY-MM-DD): %w", err)
	}

	day := req.DateDay
	if day == "" {
		day = fmt.Sprintf("%02d", parsedDate.Day())
	}
	month := req.DateMonth
	if month == "" {
		month = strings.ToUpper(parsedDate.Format("Jan"))
	}
	year := req.DateYear
	if year == "" {
		year = fmt.Sprintf("%d", parsedDate.Year())
	}
	fullText := req.DateFullText
	if fullText == "" {
		fullText = parsedDate.Format("02 January 2006")
	}

	catVariant := req.CategoryVariant
	if catVariant == "" {
		catVariant = "primary"
	}

	locType := req.LocationType
	if locType == "" {
		locType = "offline"
	}

	status := req.Status
	if status == "" {
		status = "published"
	}

	fee := req.Fee
	if fee == "" {
		if req.IsFree {
			fee = "Gratis"
		} else {
			fee = fmt.Sprintf("Rp %d", req.Price)
		}
	}

	var paymentInfoJSON, customFieldsJSON, speakersJSON, rundownJSON, benefitsJSON, prerequisitesJSON, contactJSON []byte
	if req.PaymentInfo != nil {
		paymentInfoJSON, _ = json.Marshal(req.PaymentInfo)
	} else {
		paymentInfoJSON = []byte("{}")
	}
	if req.CustomFieldsSchema != nil {
		customFieldsJSON, _ = json.Marshal(req.CustomFieldsSchema)
	} else {
		customFieldsJSON = []byte("[]")
	}
	if req.Speakers != nil {
		speakersJSON, _ = json.Marshal(req.Speakers)
	} else {
		speakersJSON = []byte("[]")
	}
	if req.Rundown != nil {
		rundownJSON, _ = json.Marshal(req.Rundown)
	} else {
		rundownJSON = []byte("[]")
	}
	if req.Benefits != nil {
		benefitsJSON, _ = json.Marshal(req.Benefits)
	} else {
		benefitsJSON = []byte("[]")
	}
	if req.Prerequisites != nil {
		prerequisitesJSON, _ = json.Marshal(req.Prerequisites)
	} else {
		prerequisitesJSON = []byte("[]")
	}
	if req.ContactPerson != nil {
		contactJSON, _ = json.Marshal(req.ContactPerson)
	}

	eventID := slug
	if eventID == "" {
		eventID = fmt.Sprintf("event-%d", time.Now().Unix())
	}

	now := time.Now()
	event := &domain.Event{
		ID:                   eventID,
		Title:                req.Title,
		Slug:                 slug,
		Description:          req.Description,
		LongDescription:      req.LongDescription,
		SeriesID:             req.SeriesID,
		SeriesName:           req.SeriesName,
		CategoryName:         req.CategoryName,
		CategoryVariant:      catVariant,
		EventDate:            req.EventDate,
		DateDay:              day,
		DateMonth:            month,
		DateYear:             year,
		DateFullText:         fullText,
		Time:                 req.Time,
		LocationName:         req.LocationName,
		LocationRoom:         req.LocationRoom,
		LocationAddress:      req.LocationAddress,
		LocationType:         locType,
		CoverImage:           req.CoverImage,
		QuotaTotal:           req.QuotaTotal,
		QuotaFilled:          0,
		QuotaStatus:          "open",
		QuotaStatusLabel:     "Pendaftaran Dibuka",
		Featured:             req.Featured,
		IsFree:               req.IsFree,
		Price:                req.Price,
		PaymentInfo:          paymentInfoJSON,
		RequiresApproval:     req.RequiresApproval,
		CustomFieldsSchema:   customFieldsJSON,
		Speakers:             speakersJSON,
		Rundown:              rundownJSON,
		Benefits:             benefitsJSON,
		Prerequisites:        prerequisitesJSON,
		TargetAudience:       req.TargetAudience,
		RegistrationDeadline: req.RegistrationDeadline,
		Fee:                  fee,
		ContactPerson:        contactJSON,
		Status:               status,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := s.eventRepo.Create(ctx, event); err != nil {
		return nil, err
	}

	s.logAudit(ctx, &actorID, "CREATE_EVENT", event.ID, map[string]interface{}{
		"title": event.Title,
		"slug":  event.Slug,
	})

	return event, nil
}

func (s *EventService) GetEventByIDOrSlug(ctx context.Context, idOrSlug string) (*domain.Event, error) {
	e, err := s.eventRepo.GetByID(ctx, idOrSlug)
	if err == nil {
		return e, nil
	}
	return s.eventRepo.GetBySlug(ctx, idOrSlug)
}

func (s *EventService) ListEvents(ctx context.Context, filter domain.EventFilter) (*domain.EventListResponse, error) {
	events, total, err := s.eventRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	return &domain.EventListResponse{
		Events: events,
		Pagination: domain.EventPagination{
			Page:       page,
			Limit:      limit,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *EventService) UpdateEvent(ctx context.Context, actorID uuid.UUID, id string, req *domain.UpdateEventRequest) (*domain.Event, error) {
	event, err := s.eventRepo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}

	s.logAudit(ctx, &actorID, "UPDATE_EVENT", id, req)
	return event, nil
}

func (s *EventService) DeleteEvent(ctx context.Context, actorID uuid.UUID, id string) error {
	if err := s.eventRepo.Delete(ctx, id); err != nil {
		return err
	}
	s.logAudit(ctx, &actorID, "DELETE_EVENT", id, nil)
	return nil
}

// ----------------- Registration Business Logic -----------------

func (s *EventService) Register(ctx context.Context, userID uuid.UUID, eventID string, req *domain.RegisterEventRequest) (*domain.EventRegistration, error) {
	event, err := s.GetEventByIDOrSlug(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("agenda tidak ditemukan: %w", err)
	}

	if event.Status != "published" {
		return nil, errors.New("agenda tidak menerima pendaftaran saat ini")
	}

	// 1. Cek tanggal event (apakah sudah berlalu)
	eventTime, err := time.Parse("2006-01-02", event.EventDate)
	if err == nil {
		today := time.Now().Truncate(24 * time.Hour)
		if eventTime.Before(today) {
			return nil, errors.New("pendaftaran ditutup karena acara telah selesai")
		}
	}

	// 2. Anti-double registration check (1 akun = 1 tiket per event)
	existing, err := s.eventRepo.GetUserRegistration(ctx, event.ID, userID)
	if err == nil && existing != nil {
		if existing.Status == "cancelled" {
			// Boleh daftar ulang jika sebelumnya membatalkan
		} else {
			return nil, errors.New("Anda sudah terdaftar pada agenda ini. Cek tiket Anda di menu Tiket Saya.")
		}
	}

	// 3. Kuota Check
	if event.QuotaFilled >= event.QuotaTotal {
		return nil, errors.New("kuota pendaftaran agenda ini sudah penuh")
	}

	// 4. Tentukan Initial Status:
	// - Jika GRATIS dan TIDAK butuh approval manual: AUTO-APPROVE INSTAN!
	// - Jika BERBAYAR atau butuh kurasi/review: PENDING_REVIEW
	status := "pending_review"
	if event.IsFree && !event.RequiresApproval {
		status = "approved"
	}

	// 5. Generate unique registration ticket code
	ticketCode := generateRandomCode("UCH-EVT")

	var answersJSON, filesJSON []byte
	if req.Answers != nil {
		answersJSON, _ = json.Marshal(req.Answers)
	} else {
		answersJSON = []byte("{}")
	}
	if req.UploadedFiles != nil {
		filesJSON, _ = json.Marshal(req.UploadedFiles)
	} else {
		filesJSON = []byte("[]")
	}

	var idNum, notes *string
	if req.IdentityNumber != "" {
		idNum = &req.IdentityNumber
	}
	if req.Notes != "" {
		notes = &req.Notes
	}

	now := time.Now()
	regID := uuid.New().String()

	reg := &domain.EventRegistration{
		ID:               regID,
		RegistrationCode: ticketCode,
		EventID:          event.ID,
		EventTitle:       event.Title,
		EventSlug:        event.Slug,
		EventDate:        event.EventDate,
		EventTime:        event.Time,
		EventLocation:    event.LocationName,
		UserID:           userID,
		FullName:         req.FullName,
		IdentityNumber:   idNum,
		Institution:      req.Institution,
		Email:            req.Email,
		Phone:            req.Phone,
		Notes:            notes,
		Answers:          answersJSON,
		UploadedFiles:    filesJSON,
		PaymentProofURL:  req.PaymentProofURL,
		Status:           status,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.eventRepo.Register(ctx, reg); err != nil {
		return nil, err
	}

	// 6. Cadangkan kuota (increment filled)
	_ = s.eventRepo.IncrementQuotaFilled(ctx, event.ID, 1)

	// 7. Audit log
	s.logAudit(ctx, &userID, "REGISTER_EVENT", reg.ID, map[string]interface{}{
		"event_id":          event.ID,
		"event_title":       event.Title,
		"registration_code": ticketCode,
		"status":            status,
	})

	// 8. Realtime notifications
	if s.notifier != nil {
		s.notifier.SendToRole("admin", "new_event_registration", map[string]interface{}{
			"registration_id": reg.ID,
			"event_title":     event.Title,
			"full_name":       req.FullName,
			"status":          status,
		})
		s.notifier.SendToUser(userID, "event_registration_success", map[string]interface{}{
			"registration_id": reg.ID,
			"ticket_code":     ticketCode,
			"event_title":     event.Title,
			"status":          status,
		})
	}

	return reg, nil
}

func (s *EventService) GetMyRegistrations(ctx context.Context, userID uuid.UUID, page, limit int) (*domain.RegistrationListResponse, error) {
	filter := domain.RegistrationFilter{
		UserID: &userID,
		Page:   page,
		Limit:  limit,
	}
	regs, total, err := s.eventRepo.ListRegistrations(ctx, filter)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 20
	}
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	return &domain.RegistrationListResponse{
		Registrations: regs,
		Pagination: domain.EventPagination{
			Page:       page,
			Limit:      limit,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *EventService) GetMyRegistrationForEvent(ctx context.Context, userID uuid.UUID, eventIDOrSlug string) (*domain.EventRegistration, error) {
	event, err := s.GetEventByIDOrSlug(ctx, eventIDOrSlug)
	if err != nil {
		return nil, err
	}
	return s.eventRepo.GetUserRegistration(ctx, event.ID, userID)
}

func (s *EventService) ListRegistrations(ctx context.Context, filter domain.RegistrationFilter) (*domain.RegistrationListResponse, error) {
	regs, total, err := s.eventRepo.ListRegistrations(ctx, filter)
	if err != nil {
		return nil, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 25
	}
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	return &domain.RegistrationListResponse{
		Registrations: regs,
		Pagination: domain.EventPagination{
			Page:       filter.Page,
			Limit:      limit,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *EventService) UpdateRegistrationStatus(ctx context.Context, actorID uuid.UUID, id string, req *domain.UpdateRegistrationStatusRequest) (*domain.EventRegistration, error) {
	existing, err := s.eventRepo.GetRegistrationByID(ctx, id)
	if err != nil {
		return nil, err
	}

	oldStatus := existing.Status
	newStatus := req.Status

	if err := s.eventRepo.UpdateRegistrationStatus(ctx, id, newStatus, req.AdminNotes, &actorID); err != nil {
		return nil, err
	}

	// Jika ditolak/dibatalkan dari yang sebelumnya approved, lepaskan kuota
	if oldStatus == "approved" && (newStatus == "rejected" || newStatus == "cancelled") {
		_ = s.eventRepo.IncrementQuotaFilled(ctx, existing.EventID, -1)
	} else if oldStatus != "approved" && newStatus == "approved" {
		// Dari pending ke approved
	}

	updated, err := s.eventRepo.GetRegistrationByID(ctx, id)
	if err != nil {
		return nil, err
	}

	s.logAudit(ctx, &actorID, "UPDATE_REGISTRATION_STATUS", id, map[string]interface{}{
		"old_status": oldStatus,
		"new_status": newStatus,
		"notes":      req.AdminNotes,
	})

	if s.notifier != nil {
		s.notifier.SendToUser(existing.UserID, "registration_status_changed", map[string]interface{}{
			"registration_id": existing.ID,
			"event_title":     existing.EventTitle,
			"new_status":      newStatus,
			"admin_notes":     req.AdminNotes,
		})
	}

	return updated, nil
}

func (s *EventService) ReviseRegistration(ctx context.Context, userID uuid.UUID, id string, req *domain.ReviseRegistrationRequest) (*domain.EventRegistration, error) {
	existing, err := s.eventRepo.GetRegistrationByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if existing.UserID != userID {
		return nil, errors.New("tidak diizinkan mengubah pendaftaran pengguna lain")
	}

	if existing.Status != "needs_revision" && existing.Status != "pending_review" {
		return nil, errors.New("pendaftaran ini tidak dalam status perlu revisi")
	}

	if err := s.eventRepo.ReviseRegistration(ctx, id, req); err != nil {
		return nil, err
	}

	updated, err := s.eventRepo.GetRegistrationByID(ctx, id)
	if err != nil {
		return nil, err
	}

	s.logAudit(ctx, &userID, "REVISE_REGISTRATION", id, nil)

	if s.notifier != nil {
		s.notifier.SendToRole("admin", "registration_revised", map[string]interface{}{
			"registration_id": id,
			"event_title":     existing.EventTitle,
			"full_name":       existing.FullName,
		})
	}

	return updated, nil
}

func (s *EventService) CheckIn(ctx context.Context, actorID uuid.UUID, code string) (*domain.EventRegistration, error) {
	trimmed := strings.TrimSpace(code)
	reg, err := s.eventRepo.CheckIn(ctx, trimmed)
	if err != nil {
		return nil, fmt.Errorf("tiket tidak valid atau tidak ditemukan: %w", err)
	}

	s.logAudit(ctx, &actorID, "CHECKIN_EVENT", reg.ID, map[string]interface{}{
		"registration_code": reg.RegistrationCode,
		"event_title":       reg.EventTitle,
		"full_name":         reg.FullName,
	})

	return reg, nil
}
