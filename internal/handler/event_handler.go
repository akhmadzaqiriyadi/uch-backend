package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"gozaq/internal/domain"
	"gozaq/internal/handler/middleware"
	"gozaq/internal/service"
	"gozaq/pkg/response"
	"gozaq/pkg/validator"
)

type EventHandler struct {
	eventService *service.EventService
}

func NewEventHandler(eventService *service.EventService) *EventHandler {
	return &EventHandler{eventService: eventService}
}

// ----------------- Public Endpoints -----------------

// ListEvents handles GET /api/v1/events
func (h *EventHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	upcomingOnly := q.Get("upcoming") == "true"
	pastOnly := q.Get("past") == "true"

	filter := domain.EventFilter{
		Category:     q.Get("category"),
		SeriesID:     q.Get("series_id"),
		Status:       q.Get("status"),
		Search:       q.Get("search"),
		UpcomingOnly: upcomingOnly,
		PastOnly:     pastOnly,
		Page:         page,
		Limit:        limit,
	}

	res, err := h.eventService.ListEvents(r.Context(), filter)
	if err != nil {
		response.InternalServerError(w, "Gagal memuat daftar agenda", err.Error())
		return
	}

	response.OK(w, "Daftar agenda berhasil diambil", res)
}

// GetEvent handles GET /api/v1/events/{id}
func (h *EventHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	idOrSlug := chi.URLParam(r, "id")
	event, err := h.eventService.GetEventByIDOrSlug(r.Context(), idOrSlug)
	if err != nil {
		response.NotFound(w, "Agenda tidak ditemukan")
		return
	}

	response.OK(w, "Detail agenda berhasil dimuat", event)
}

// ----------------- User Endpoints (Protected) -----------------

// Register handles POST /api/v1/events/{id}/register
func (h *EventHandler) Register(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		response.Unauthorized(w, "Wajib login untuk mendaftar agenda")
		return
	}

	eventID := chi.URLParam(r, "id")
	var req domain.RegisterEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Format data pendaftaran tidak valid", err.Error())
		return
	}

	if errs := validator.ValidateStruct(req); len(errs) > 0 {
		response.UnprocessableEntity(w, "Mohon lengkapi formulir pendaftaran dengan benar", errs)
		return
	}

	reg, err := h.eventService.Register(r.Context(), userID, eventID, &req)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "sudah terdaftar") {
			response.Error(w, http.StatusConflict, "ALREADY_REGISTERED", errMsg, nil)
			return
		}
		if strings.Contains(errMsg, "penuh") {
			response.Error(w, http.StatusConflict, "QUOTA_FULL", errMsg, nil)
			return
		}
		if strings.Contains(errMsg, "ditutup") || strings.Contains(errMsg, "selesai") {
			response.Error(w, http.StatusBadRequest, "EVENT_CLOSED", errMsg, nil)
			return
		}
		response.InternalServerError(w, "Gagal memproses pendaftaran", errMsg)
		return
	}

	msg := "Pendaftaran agenda berhasil dikonfirmasi"
	if reg.Status == "pending_review" {
		msg = "Pendaftaran berhasil dikirim, menunggu verifikasi berkas/pembayaran oleh panitia"
	}

	response.Created(w, msg, reg)
}

// GetMyRegistrations handles GET /api/v1/my-event-registrations
func (h *EventHandler) GetMyRegistrations(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		response.Unauthorized(w, "Wajib login untuk melihat tiket agenda")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	res, err := h.eventService.GetMyRegistrations(r.Context(), userID, page, limit)
	if err != nil {
		response.InternalServerError(w, "Gagal memuat tiket saya", err.Error())
		return
	}

	response.OK(w, "Daftar tiket saya berhasil diambil", res)
}

// GetMyRegistrationForEvent handles GET /api/v1/events/{id}/my-registration
func (h *EventHandler) GetMyRegistrationForEvent(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		response.Unauthorized(w, "Wajib login")
		return
	}

	eventID := chi.URLParam(r, "id")
	reg, err := h.eventService.GetMyRegistrationForEvent(r.Context(), userID, eventID)
	if err != nil {
		response.NotFound(w, "Belum terdaftar pada agenda ini")
		return
	}

	response.OK(w, "Status pendaftaran Anda", reg)
}

// ReviseRegistration handles POST /api/v1/events/registrations/{id}/revise
func (h *EventHandler) ReviseRegistration(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		response.Unauthorized(w, "Wajib login")
		return
	}

	id := chi.URLParam(r, "id")
	var req domain.ReviseRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Format data revisi tidak valid", err.Error())
		return
	}

	updated, err := h.eventService.ReviseRegistration(r.Context(), userID, id, &req)
	if err != nil {
		response.BadRequest(w, "Gagal mengirimkan revisi pendaftaran", err.Error())
		return
	}

	response.OK(w, "Revisi berkas pendaftaran berhasil dikirim untuk diverifikasi ulang", updated)
}

// ----------------- Admin Endpoints -----------------

// CreateEvent handles POST /api/v1/events
func (h *EventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	actorID, err := middleware.GetUserID(r.Context())
	if err != nil {
		response.Unauthorized(w, "Autentikasi diperlukan")
		return
	}

	var req domain.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Format data agenda tidak valid", err.Error())
		return
	}

	if errs := validator.ValidateStruct(req); len(errs) > 0 {
		response.UnprocessableEntity(w, "Mohon lengkapi formulir agenda", errs)
		return
	}

	event, err := h.eventService.CreateEvent(r.Context(), actorID, &req)
	if err != nil {
		response.InternalServerError(w, "Gagal membuat agenda baru", err.Error())
		return
	}

	response.Created(w, "Agenda berhasil dipublikasikan", event)
}

// UpdateEvent handles PUT /api/v1/events/{id}
func (h *EventHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	actorID, err := middleware.GetUserID(r.Context())
	if err != nil {
		response.Unauthorized(w, "Autentikasi diperlukan")
		return
	}

	id := chi.URLParam(r, "id")
	var req domain.UpdateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Format data pembaruan tidak valid", err.Error())
		return
	}

	event, err := h.eventService.UpdateEvent(r.Context(), actorID, id, &req)
	if err != nil {
		response.InternalServerError(w, "Gagal memperbarui agenda", err.Error())
		return
	}

	response.OK(w, "Agenda berhasil diperbarui", event)
}

// DeleteEvent handles DELETE /api/v1/events/{id}
func (h *EventHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	actorID, err := middleware.GetUserID(r.Context())
	if err != nil {
		response.Unauthorized(w, "Autentikasi diperlukan")
		return
	}

	id := chi.URLParam(r, "id")
	if err := h.eventService.DeleteEvent(r.Context(), actorID, id); err != nil {
		response.InternalServerError(w, "Gagal menghapus agenda", err.Error())
		return
	}

	response.OK(w, "Agenda berhasil dihapus", nil)
}

// ListRegistrations handles GET /api/v1/events/{id}/registrations and GET /api/v1/events/registrations
func (h *EventHandler) ListRegistrations(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "id")
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	filter := domain.RegistrationFilter{
		EventID: eventID,
		Status:  q.Get("status"),
		Search:  q.Get("search"),
		Page:    page,
		Limit:   limit,
	}

	res, err := h.eventService.ListRegistrations(r.Context(), filter)
	if err != nil {
		response.InternalServerError(w, "Gagal memuat data pendaftar", err.Error())
		return
	}

	response.OK(w, "Daftar pendaftar berhasil diambil", res)
}

// UpdateRegistrationStatus handles PATCH /api/v1/events/registrations/{id}/status
func (h *EventHandler) UpdateRegistrationStatus(w http.ResponseWriter, r *http.Request) {
	actorID, err := middleware.GetUserID(r.Context())
	if err != nil {
		response.Unauthorized(w, "Autentikasi diperlukan")
		return
	}

	id := chi.URLParam(r, "id")
	var req domain.UpdateRegistrationStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Format data status tidak valid", err.Error())
		return
	}

	if errs := validator.ValidateStruct(req); len(errs) > 0 {
		response.UnprocessableEntity(w, "Status tidak valid", errs)
		return
	}

	updated, err := h.eventService.UpdateRegistrationStatus(r.Context(), actorID, id, &req)
	if err != nil {
		response.InternalServerError(w, "Gagal memperbarui status pendaftaran", err.Error())
		return
	}

	response.OK(w, "Status pendaftaran berhasil diperbarui", updated)
}

// CheckIn handles POST /api/v1/events/checkin
func (h *EventHandler) CheckIn(w http.ResponseWriter, r *http.Request) {
	actorID, err := middleware.GetUserID(r.Context())
	if err != nil {
		response.Unauthorized(w, "Autentikasi panitia diperlukan")
		return
	}

	var req domain.EventCheckInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Kode tiket tidak valid", err.Error())
		return
	}

	if req.RegistrationCode == "" {
		response.BadRequest(w, "Kode tiket wajib diisi", nil)
		return
	}

	reg, err := h.eventService.CheckIn(r.Context(), actorID, req.RegistrationCode)
	if err != nil {
		response.BadRequest(w, "Gagal check-in", err.Error())
		return
	}

	response.OK(w, fmt.Sprintf("Check-in berhasil! Selamat datang %s", reg.FullName), reg)
}

// ExportRegistrationsCSV handles GET /api/v1/events/{id}/export
func (h *EventHandler) ExportRegistrationsCSV(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "id")
	filter := domain.RegistrationFilter{
		EventID: eventID,
		Limit:   1000,
	}

	res, err := h.eventService.ListRegistrations(r.Context(), filter)
	if err != nil {
		response.InternalServerError(w, "Gagal mengekspor pendaftar", err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=peserta-event-%s.csv", eventID))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write CSV header
	_ = writer.Write([]string{
		"Kode Tiket", "Nama Lengkap", "NPM / Identitas", "Institusi",
		"Email", "No. WhatsApp", "Status", "Bukti Pembayaran",
		"Tanggal Daftar", "Waktu Kehadiran", "Catatan",
	})

	for _, reg := range res.Registrations {
		idNum := ""
		if reg.IdentityNumber != nil {
			idNum = *reg.IdentityNumber
		}
		proof := ""
		if reg.PaymentProofURL != nil {
			proof = *reg.PaymentProofURL
		}
		notes := ""
		if reg.Notes != nil {
			notes = *reg.Notes
		}
		attendedAt := ""
		if reg.AttendedAt != nil {
			attendedAt = reg.AttendedAt.Format("02-01-2006 15:04:05")
		}

		_ = writer.Write([]string{
			reg.RegistrationCode,
			reg.FullName,
			idNum,
			reg.Institution,
			reg.Email,
			reg.Phone,
			reg.Status,
			proof,
			reg.CreatedAt.Format("02-01-2006 15:04:05"),
			attendedAt,
			notes,
		})
	}
}
