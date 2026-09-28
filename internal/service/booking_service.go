package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"

	"gozaq/internal/domain"
)

type RealtimeNotifier interface {
	SendToUser(userID uuid.UUID, event string, payload any)
	SendToRole(role string, event string, payload any)
	Broadcast(event string, payload any)
}

type BookingService struct {
	bookingRepo domain.BookingRepository
	roomRepo    domain.RoomRepository
	userRepo    domain.UserRepository
	auditRepo   domain.AuditRepository
	notifier    RealtimeNotifier
}

func NewBookingService(
	bookingRepo domain.BookingRepository,
	roomRepo domain.RoomRepository,
	userRepo domain.UserRepository,
	auditRepo domain.AuditRepository,
) *BookingService {
	return &BookingService{
		bookingRepo: bookingRepo,
		roomRepo:    roomRepo,
		userRepo:    userRepo,
		auditRepo:   auditRepo,
	}
}

func (s *BookingService) SetNotifier(notifier RealtimeNotifier) {
	s.notifier = notifier
}

func (s *BookingService) generateBookingCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	code := n.Int64() + 100000
	year := time.Now().Year()
	return fmt.Sprintf("UCH-%d-%d", year, code)
}

func (s *BookingService) CreateBooking(ctx context.Context, userID uuid.UUID, req *domain.CreateBookingRequest) (*domain.Booking, error) {
	// 1. Verify Room exists
	room, err := s.roomRepo.GetByID(ctx, req.RoomID)
	if err != nil {
		return nil, fmt.Errorf("ruangan tidak ditemukan: %w", err)
	}

	roomName := req.RoomName
	if roomName == "" {
		roomName = room.Name
	}

	// 2. Check for slot conflict with existing approved/active bookings
	conflict, err := s.bookingRepo.CheckConflict(ctx, room.ID, req.BookingDate, req.StartTime, req.EndTime, "")
	if err != nil {
		return nil, fmt.Errorf("gagal memeriksa ketersediaan jadwal ruangan: %w", err)
	}
	if conflict != nil {
		return nil, fmt.Errorf("%w: jadwal ruangan %s pada tanggal %s pukul %s - %s telah terisi/disetujui untuk kegiatan '%s' (Kode: %s)",
			domain.ErrConflict, roomName, req.BookingDate, conflict.StartTime, conflict.EndTime, conflict.Purpose, conflict.ID)
	}

	// 3. Build booking model
	bookingID := s.generateBookingCode()
	now := time.Now().UTC()

	booking := &domain.Booking{
		ID:            bookingID,
		UserID:        userID,
		RoomID:        room.ID,
		RoomName:      roomName,
		ApplicantName: req.ApplicantName,
		ApplicantRole: req.ApplicantRole,
		IDNumber:      req.IDNumber,
		Prodi:         req.Prodi,
		Purpose:       req.Purpose,
		Audience:      req.Audience,
		BookingDate:   req.BookingDate,
		StartTime:     req.StartTime,
		EndTime:       req.EndTime,
		Status:        "pending",
		AdminNotes:    nil,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.bookingRepo.Create(ctx, booking); err != nil {
		return nil, fmt.Errorf("gagal membuat permohonan reservasi: %w", err)
	}

	// 3. Record Audit Log
	if s.auditRepo != nil {
		auditDetails, _ := json.Marshal(map[string]interface{}{
			"booking_id":     bookingID,
			"room_id":        room.ID,
			"room_name":      roomName,
			"applicant_name": req.ApplicantName,
			"booking_date":   req.BookingDate,
			"time_slot":      fmt.Sprintf("%s - %s", req.StartTime, req.EndTime),
		})
		_ = s.auditRepo.Create(ctx, &domain.AuditLog{
			ID:        uuid.New(),
			UserID:    &userID,
			Action:    "booking:create",
			Entity:    "bookings",
			EntityID:  bookingID,
			Details:   auditDetails,
			CreatedAt: now,
		})
	}

	if s.notifier != nil {
		s.notifier.SendToRole("admin", "booking:created", map[string]any{
			"id":             booking.ID,
			"room_name":      booking.RoomName,
			"applicant_name": booking.ApplicantName,
			"booking_date":   booking.BookingDate,
			"start_time":     booking.StartTime,
			"end_time":       booking.EndTime,
			"message":        fmt.Sprintf("Pemesanan baru dari %s untuk ruangan %s", booking.ApplicantName, booking.RoomName),
		})
	}

	return booking, nil
}

func (s *BookingService) ListBookings(ctx context.Context, filter domain.BookingFilter) (*domain.BookingListResponse, error) {
	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	bookings, total, err := s.bookingRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	if bookings == nil {
		bookings = []domain.Booking{}
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.Limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	return &domain.BookingListResponse{
		Bookings: bookings,
		Pagination: domain.BookingPagination{
			Page:       filter.Page,
			Limit:      filter.Limit,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *BookingService) GetBooking(ctx context.Context, id string) (*domain.Booking, error) {
	return s.bookingRepo.GetByID(ctx, id)
}

func (s *BookingService) UpdateBookingStatus(ctx context.Context, adminID uuid.UUID, id string, req *domain.UpdateBookingStatusRequest) (*domain.Booking, error) {
	booking, err := s.bookingRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Status == "approved" {
		conflict, err := s.bookingRepo.CheckConflict(ctx, booking.RoomID, booking.BookingDate, booking.StartTime, booking.EndTime, booking.ID)
		if err != nil {
			return nil, fmt.Errorf("gagal memeriksa bentrok jadwal ruangan: %w", err)
		}
		if conflict != nil {
			return nil, fmt.Errorf("%w: tidak dapat menyetujui, ruangan %s pada tanggal %s jam %s - %s sudah disetujui untuk reservasi %s (%s)",
				domain.ErrConflict, booking.RoomName, booking.BookingDate, conflict.StartTime, conflict.EndTime, conflict.ID, conflict.ApplicantName)
		}
	}

	if err := s.bookingRepo.UpdateStatus(ctx, id, req.Status, req.Notes); err != nil {
		return nil, err
	}

	booking.Status = req.Status
	booking.AdminNotes = req.Notes
	booking.UpdatedAt = time.Now().UTC()

	// Record Audit Log
	if s.auditRepo != nil {
		auditDetails, _ := json.Marshal(map[string]interface{}{
			"booking_id": id,
			"old_status": booking.Status,
			"new_status": req.Status,
			"notes":      req.Notes,
		})
		_ = s.auditRepo.Create(ctx, &domain.AuditLog{
			ID:        uuid.New(),
			UserID:    &adminID,
			Action:    "booking:update_status",
			Entity:    "bookings",
			EntityID:  id,
			Details:   auditDetails,
			CreatedAt: time.Now().UTC(),
		})
	}

	if s.notifier != nil {
		notesStr := ""
		if req.Notes != nil {
			notesStr = *req.Notes
		}

		if req.Status == "approved" {
			s.notifier.SendToUser(booking.UserID, "booking:approved", map[string]any{
				"id":           booking.ID,
				"room_name":    booking.RoomName,
				"booking_date": booking.BookingDate,
				"start_time":   booking.StartTime,
				"end_time":     booking.EndTime,
				"status":       "approved",
				"message":      fmt.Sprintf("Pemesanan ruangan %s Anda telah DISETUJUI oleh Admin UCH!", booking.RoomName),
			})
		} else if req.Status == "rejected" {
			s.notifier.SendToUser(booking.UserID, "booking:rejected", map[string]any{
				"id":           booking.ID,
				"room_name":    booking.RoomName,
				"booking_date": booking.BookingDate,
				"status":       "rejected",
				"reason":       notesStr,
				"message":      fmt.Sprintf("Pemesanan ruangan %s Anda DITOLAK. Alasan: %s", booking.RoomName, notesStr),
			})
		}

		s.notifier.SendToRole("admin", "booking:updated", map[string]any{
			"id":        booking.ID,
			"room_name": booking.RoomName,
			"status":    req.Status,
			"notes":     notesStr,
		})
	}

	return booking, nil
}

func (s *BookingService) CancelBooking(ctx context.Context, userID uuid.UUID, id string, isAdmin bool) error {
	booking, err := s.bookingRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// If not admin, ensure the booking belongs to this user
	if !isAdmin && booking.UserID != userID {
		return domain.ErrForbidden
	}

	cancelNotes := "Dibatalkan oleh pemohon"
	if isAdmin {
		cancelNotes = "Dibatalkan oleh pengelola UCH"
	}

	if err := s.bookingRepo.UpdateStatus(ctx, id, "cancelled", &cancelNotes); err != nil {
		return err
	}

	if s.auditRepo != nil {
		auditDetails, _ := json.Marshal(map[string]interface{}{
			"booking_id": id,
			"cancelled_by": userID.String(),
		})
		_ = s.auditRepo.Create(ctx, &domain.AuditLog{
			ID:        uuid.New(),
			UserID:    &userID,
			Action:    "booking:cancel",
			Entity:    "bookings",
			EntityID:  id,
			Details:   auditDetails,
			CreatedAt: time.Now().UTC(),
		})
	}

	return nil
}

func (s *BookingService) DeleteBooking(ctx context.Context, id string) error {
	return s.bookingRepo.Delete(ctx, id)
}

func (s *BookingService) AdminCheckIn(ctx context.Context, adminID uuid.UUID, bookingCode string) (*domain.Booking, error) {
	bookingCode = strings.TrimSpace(bookingCode)
	booking, err := s.bookingRepo.GetByID(ctx, bookingCode)
	if err != nil {
		return nil, fmt.Errorf("kode reservasi '%s' tidak ditemukan: %w", bookingCode, domain.ErrNotFound)
	}

	if booking.Status == "completed" {
		return nil, fmt.Errorf("reservasi %s sudah check-in sebelumnya", booking.ID)
	}

	if booking.Status != "approved" {
		return nil, fmt.Errorf("reservasi %s belum disetujui (status saat ini: %s)", booking.ID, booking.Status)
	}

	notes := fmt.Sprintf("Check-in presensi diverifikasi oleh administrator pada %s", time.Now().Format("02 Jan 2006 15:04 WIB"))
	if err := s.bookingRepo.UpdateStatus(ctx, booking.ID, "completed", &notes); err != nil {
		return nil, err
	}

	booking.Status = "completed"
	booking.AdminNotes = &notes
	booking.UpdatedAt = time.Now().UTC()

	if s.auditRepo != nil {
		auditDetails, _ := json.Marshal(map[string]interface{}{
			"booking_id":     booking.ID,
			"room_name":      booking.RoomName,
			"applicant_name": booking.ApplicantName,
			"checkin_type":   "admin_scan",
			"verified_by":    adminID.String(),
		})
		_ = s.auditRepo.Create(ctx, &domain.AuditLog{
			ID:        uuid.New(),
			UserID:    &adminID,
			Action:    "booking:checkin",
			Entity:    "bookings",
			EntityID:  booking.ID,
			Details:   auditDetails,
			CreatedAt: time.Now().UTC(),
		})
	}

	if s.notifier != nil {
		s.notifier.SendToUser(booking.UserID, "booking:checked_in", map[string]any{
			"id":        booking.ID,
			"room_name": booking.RoomName,
			"status":    "completed",
			"message":   fmt.Sprintf("Presensi check-in untuk ruangan %s berhasil diverifikasi oleh Admin!", booking.RoomName),
		})
		s.notifier.SendToRole("admin", "booking:updated", map[string]any{
			"id":        booking.ID,
			"room_name": booking.RoomName,
			"status":    "completed",
		})
	}

	return booking, nil
}

func (s *BookingService) SelfCheckIn(ctx context.Context, userID uuid.UUID, req *domain.SelfCheckInRequest) (*domain.Booking, error) {
	// 1. Direct code check-in if booking_code is provided
	if req.BookingCode != "" {
		code := strings.TrimSpace(req.BookingCode)
		booking, err := s.bookingRepo.GetByID(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("kode reservasi tidak ditemukan: %w", domain.ErrNotFound)
		}
		if booking.UserID != userID {
			return nil, domain.ErrForbidden
		}
		if booking.Status == "completed" {
			return nil, fmt.Errorf("reservasi %s sudah melakukan check-in sebelumnya", booking.ID)
		}
		if booking.Status != "approved" {
			return nil, fmt.Errorf("reservasi belum disetujui (status: %s)", booking.Status)
		}

		notes := fmt.Sprintf("Check-in mandiri oleh pemohon via scan QR pada %s", time.Now().Format("02 Jan 2006 15:04 WIB"))
		if err := s.bookingRepo.UpdateStatus(ctx, booking.ID, "completed", &notes); err != nil {
			return nil, err
		}
		booking.Status = "completed"
		booking.AdminNotes = &notes
		booking.UpdatedAt = time.Now().UTC()

		if s.auditRepo != nil {
			auditDetails, _ := json.Marshal(map[string]interface{}{
				"booking_id":   booking.ID,
				"room_name":    booking.RoomName,
				"checkin_type": "user_self_scan",
			})
			_ = s.auditRepo.Create(ctx, &domain.AuditLog{
				ID:        uuid.New(),
				UserID:    &userID,
				Action:    "booking:self_checkin",
				Entity:    "bookings",
				EntityID:  booking.ID,
				Details:   auditDetails,
				CreatedAt: time.Now().UTC(),
			})
		}

		if s.notifier != nil {
			s.notifier.SendToUser(booking.UserID, "booking:checked_in", map[string]any{
				"id":        booking.ID,
				"room_name": booking.RoomName,
				"status":    "completed",
				"message":   fmt.Sprintf("Self check-in berhasil untuk ruangan %s!", booking.RoomName),
			})
			s.notifier.SendToRole("admin", "booking:checked_in", map[string]any{
				"id":             booking.ID,
				"applicant_name": booking.ApplicantName,
				"room_name":      booking.RoomName,
				"status":         "completed",
				"message":        fmt.Sprintf("%s telah check-in di ruangan %s", booking.ApplicantName, booking.RoomName),
			})
		}

		return booking, nil
	}

	// 2. Room QR check-in: Match user's active approved booking for this room
	roomID := strings.TrimPrefix(strings.TrimSpace(req.RoomID), "UCH-ROOM:")
	bookings, _, err := s.bookingRepo.List(ctx, domain.BookingFilter{
		UserID: &userID,
		RoomID: roomID,
		Status: "approved",
		Limit:  10,
	})
	if err != nil {
		return nil, err
	}

	if len(bookings) == 0 {
		compBookings, _, _ := s.bookingRepo.List(ctx, domain.BookingFilter{
			UserID: &userID,
			RoomID: roomID,
			Status: "completed",
			Limit:  1,
		})
		if len(compBookings) > 0 {
			return nil, fmt.Errorf("Anda sudah melakukan check-in untuk ruangan ini (%s)", compBookings[0].ID)
		}
		return nil, fmt.Errorf("tidak ditemukan reservasi aktif yang disetujui untuk ruangan ini")
	}

	target := bookings[0]
	notes := fmt.Sprintf("Check-in mandiri di pintu ruangan via scan QR pada %s", time.Now().Format("02 Jan 2006 15:04 WIB"))
	if err := s.bookingRepo.UpdateStatus(ctx, target.ID, "completed", &notes); err != nil {
		return nil, err
	}

	target.Status = "completed"
	target.AdminNotes = &notes
	target.UpdatedAt = time.Now().UTC()

	if s.auditRepo != nil {
		auditDetails, _ := json.Marshal(map[string]interface{}{
			"booking_id":   target.ID,
			"room_id":      roomID,
			"room_name":    target.RoomName,
			"checkin_type": "user_room_qr_scan",
		})
		_ = s.auditRepo.Create(ctx, &domain.AuditLog{
			ID:        uuid.New(),
			UserID:    &userID,
			Action:    "booking:self_checkin",
			Entity:    "bookings",
			EntityID:  target.ID,
			Details:   auditDetails,
			CreatedAt: time.Now().UTC(),
		})
	}

	if s.notifier != nil {
		s.notifier.SendToUser(target.UserID, "booking:checked_in", map[string]any{
			"id":        target.ID,
			"room_name": target.RoomName,
			"status":    "completed",
			"message":   fmt.Sprintf("Self check-in pintu ruangan %s berhasil!", target.RoomName),
		})
		s.notifier.SendToRole("admin", "booking:checked_in", map[string]any{
			"id":             target.ID,
			"applicant_name": target.ApplicantName,
			"room_name":      target.RoomName,
			"status":         "completed",
			"message":        fmt.Sprintf("%s telah check-in di ruangan %s", target.ApplicantName, target.RoomName),
		})
	}

	return &target, nil
}

func (s *BookingService) GetOccupiedSlots(ctx context.Context, roomID string, date string) ([]domain.Booking, error) {
	return s.bookingRepo.GetOccupiedSlots(ctx, roomID, date)
}
