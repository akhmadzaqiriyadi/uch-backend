package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"gozaq/pkg/webpush"
)

type CompositeNotifier struct {
	realtime RealtimeNotifier
	webpush  *webpush.Service
}

func NewCompositeNotifier(realtime RealtimeNotifier, webpush *webpush.Service) *CompositeNotifier {
	return &CompositeNotifier{
		realtime: realtime,
		webpush:  webpush,
	}
}

func (c *CompositeNotifier) SendToUser(userID uuid.UUID, event string, payload any) {
	if c.realtime != nil {
		c.realtime.SendToUser(userID, event, payload)
	}
	if c.webpush != nil {
		pushPayload := formatPushPayload(event, payload)
		c.webpush.SendToUser(context.Background(), userID, pushPayload)
	}
}

func (c *CompositeNotifier) SendToRole(role string, event string, payload any) {
	if c.realtime != nil {
		c.realtime.SendToRole(role, event, payload)
	}
	if c.webpush != nil {
		pushPayload := formatPushPayload(event, payload)
		c.webpush.SendToRole(context.Background(), role, pushPayload)
	}
}

func (c *CompositeNotifier) Broadcast(event string, payload any) {
	if c.realtime != nil {
		c.realtime.Broadcast(event, payload)
	}
}

func formatPushPayload(event string, payload any) webpush.PushPayload {
	title := "Creative Hub UCH"
	message := "Ada pembaruan status peminjaman ruangan."
	url := "/my-bookings"
	bookingID := ""

	if m, ok := payload.(map[string]any); ok {
		if id, ok := m["id"].(string); ok {
			bookingID = id
		}
		if msg, ok := m["message"].(string); ok && msg != "" {
			message = msg
		}
		roomName := ""
		if rn, ok := m["room_name"].(string); ok {
			roomName = rn
		}

		switch event {
		case "booking:approved":
			title = "Pemesanan Disetujui!"
			if roomName != "" && message == "" {
				message = fmt.Sprintf("Pemesanan ruangan %s Anda telah disetujui Admin.", roomName)
			}
			url = "/my-bookings"
		case "booking:rejected":
			title = "Pemesanan Ditolak"
			if roomName != "" && message == "" {
				message = fmt.Sprintf("Pemesanan ruangan %s Anda ditolak.", roomName)
			}
			url = "/my-bookings"
		case "booking:created":
			if appName, ok := m["applicant_name"].(string); ok && appName != "" {
				title = "Permohonan Ruangan Baru"
				message = fmt.Sprintf("%s mengajukan peminjaman ruangan %s.", appName, roomName)
				url = "/admin"
			} else {
				title = "Permohonan Berhasil Diajukan"
				message = fmt.Sprintf("Permohonan ruangan %s berhasil diajukan dan sedang ditinjau.", roomName)
				url = "/my-bookings"
			}
		case "booking:checked_in":
			title = "Presensi Selesai"
			if message == "" {
				message = fmt.Sprintf("Presensi check-in untuk %s berhasil diverifikasi.", roomName)
			}
		case "booking:updated":
			title = "Pembaruan Status Booking"
			url = "/admin"
		}
	}

	return webpush.PushPayload{
		Title:     title,
		Message:   message,
		Body:      message,
		URL:       url,
		BookingID: bookingID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}
