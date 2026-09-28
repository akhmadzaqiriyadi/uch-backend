package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"gozaq/internal/domain"
	"gozaq/internal/handler/middleware"
	"gozaq/internal/service"
	"gozaq/pkg/response"
	"gozaq/pkg/validator"
)

type BookingHandler struct {
	bookingService *service.BookingService
}

func NewBookingHandler(bookingService *service.BookingService) *BookingHandler {
	return &BookingHandler{bookingService: bookingService}
}

func (h *BookingHandler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		response.Unauthorized(w, "Wajib login untuk mengajukan reservasi ruangan")
		return
	}

	var req domain.CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Format data reservasi tidak valid", err.Error())
		return
	}

	if errs := validator.ValidateStruct(req); len(errs) > 0 {
		response.UnprocessableEntity(w, "Mohon lengkapi formulir permohonan", errs)
		return
	}

	booking, err := h.bookingService.CreateBooking(r.Context(), userID, &req)
	if err != nil {
		response.InternalServerError(w, "Gagal membuat permohonan reservasi", err.Error())
		return
	}

	response.Created(w, "Permohonan reservasi berhasil diajukan", booking)
}

func (h *BookingHandler) ListMyBookings(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		response.Unauthorized(w, "Wajib login untuk melihat riwayat reservasi")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")

	filter := domain.BookingFilter{
		UserID: &userID,
		Status: status,
		Search: search,
		Page:   page,
		Limit:  limit,
	}

	result, err := h.bookingService.ListBookings(r.Context(), filter)
	if err != nil {
		response.InternalServerError(w, "Gagal mengambil daftar reservasi saya", err.Error())
		return
	}

	response.OK(w, "Daftar reservasi saya berhasil diambil", result)
}

func (h *BookingHandler) ListAllBookings(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}
	status := r.URL.Query().Get("status")
	roomID := r.URL.Query().Get("room_id")
	search := r.URL.Query().Get("search")

	filter := domain.BookingFilter{
		RoomID: roomID,
		Status: status,
		Search: search,
		Page:   page,
		Limit:  limit,
	}

	result, err := h.bookingService.ListBookings(r.Context(), filter)
	if err != nil {
		response.InternalServerError(w, "Gagal mengambil daftar seluruh reservasi", err.Error())
		return
	}

	response.OK(w, "Daftar reservasi berhasil diambil", result)
}

func (h *BookingHandler) GetBooking(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	booking, err := h.bookingService.GetBooking(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(w, "Data reservasi tidak ditemukan")
			return
		}
		response.InternalServerError(w, "Gagal mengambil detail reservasi", err.Error())
		return
	}

	response.OK(w, "Detail reservasi berhasil diambil", booking)
}

func (h *BookingHandler) UpdateBookingStatus(w http.ResponseWriter, r *http.Request) {
	adminID, err := middleware.GetUserID(r.Context())
	if err != nil {
		response.Unauthorized(w, "Autentikasi admin diperlukan")
		return
	}

	id := chi.URLParam(r, "id")
	var req domain.UpdateBookingStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Format data status tidak valid", err.Error())
		return
	}

	if errs := validator.ValidateStruct(req); len(errs) > 0 {
		response.UnprocessableEntity(w, "Validasi status gagal", errs)
		return
	}

	booking, err := h.bookingService.UpdateBookingStatus(r.Context(), adminID, id, &req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(w, "Data reservasi tidak ditemukan")
			return
		}
		response.InternalServerError(w, "Gagal memperbarui status reservasi", err.Error())
		return
	}

	response.OK(w, "Status reservasi berhasil diperbarui", booking)
}

func (h *BookingHandler) CancelBooking(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		response.Unauthorized(w, "Autentikasi diperlukan")
		return
	}

	id := chi.URLParam(r, "id")
	role := middleware.GetUserRole(r.Context())
	isAdmin := role == "admin" || role == "manager"

	if err := h.bookingService.CancelBooking(r.Context(), userID, id, isAdmin); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(w, "Data reservasi tidak ditemukan")
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			response.Forbidden(w, "Anda tidak berhak membatalkan reservasi ini", nil)
			return
		}
		response.InternalServerError(w, "Gagal membatalkan reservasi", err.Error())
		return
	}

	response.OK(w, "Reservasi berhasil dibatalkan", nil)
}

func (h *BookingHandler) DeleteBooking(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.bookingService.DeleteBooking(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(w, "Data reservasi tidak ditemukan")
			return
		}
		response.InternalServerError(w, "Gagal menghapus data reservasi", err.Error())
		return
	}

	response.OK(w, "Data reservasi berhasil dihapus", nil)
}

func (h *BookingHandler) AdminCheckIn(w http.ResponseWriter, r *http.Request) {
	adminID, err := middleware.GetUserID(r.Context())
	if err != nil {
		response.Unauthorized(w, "Autentikasi admin diperlukan")
		return
	}

	var req domain.AdminCheckInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Format data check-in tidak valid", err.Error())
		return
	}

	if errs := validator.ValidateStruct(req); len(errs) > 0 {
		response.UnprocessableEntity(w, "Kode reservasi wajib disertakan", errs)
		return
	}

	booking, err := h.bookingService.AdminCheckIn(r.Context(), adminID, req.BookingCode)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(w, "Kode reservasi tidak ditemukan dalam sistem")
			return
		}
		response.BadRequest(w, err.Error(), nil)
		return
	}

	response.OK(w, "Check-in reservasi berhasil diverifikasi", booking)
}

func (h *BookingHandler) SelfCheckIn(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		response.Unauthorized(w, "Wajib login untuk melakukan self check-in")
		return
	}

	var req domain.SelfCheckInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Format data QR tidak valid", err.Error())
		return
	}

	if req.RoomID == "" && req.BookingCode == "" {
		response.BadRequest(w, "Diperlukan kode reservasi atau ID ruangan dari QR", nil)
		return
	}

	booking, err := h.bookingService.SelfCheckIn(r.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.NotFound(w, "Data reservasi tidak ditemukan")
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			response.Forbidden(w, "Anda bukan pemilik dari tiket reservasi ini", nil)
			return
		}
		response.BadRequest(w, err.Error(), nil)
		return
	}

	response.OK(w, "Check-in mandiri berhasil! Selamat beraktivitas di UTY Creative Hub", booking)
}
