package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"gozaq/internal/domain"
)

type RoomService struct {
	roomRepo  domain.RoomRepository
	auditRepo domain.AuditRepository
}

func NewRoomService(roomRepo domain.RoomRepository, auditRepo domain.AuditRepository) *RoomService {
	return &RoomService{
		roomRepo:  roomRepo,
		auditRepo: auditRepo,
	}
}

func (s *RoomService) logAudit(ctx context.Context, action, entityID string, details any) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Create(ctx, &domain.AuditLog{
		ID:        uuid.New(),
		Action:    action,
		Entity:    "rooms",
		EntityID:  entityID,
		Details:   details,
		CreatedAt: time.Now().UTC(),
	})
}

func (s *RoomService) ListRooms(ctx context.Context) ([]domain.Room, error) {
	return s.roomRepo.List(ctx)
}

func (s *RoomService) GetRoom(ctx context.Context, id string) (*domain.Room, error) {
	return s.roomRepo.GetByID(ctx, id)
}

func (s *RoomService) CreateRoom(ctx context.Context, req *domain.CreateRoomRequest) (*domain.Room, error) {
	slug := req.Slug
	if slug == "" {
		slug = strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
	}
	id := req.ID
	if id == "" {
		id = slug
	}

	facJSON, err := json.Marshal(req.Facilities)
	if err != nil {
		facJSON = []byte("[]")
	}

	status := req.Status
	if status == "" {
		status = "available"
	}
	opHours := req.OperationalHours
	if opHours == "" {
		opHours = "08:00 - 21:00 WIB"
	}

	now := time.Now().UTC()
	room := &domain.Room{
		ID:               id,
		Name:             req.Name,
		Slug:             slug,
		Category:         req.Category,
		Capacity:         req.Capacity,
		Location:         req.Location,
		Description:      req.Description,
		Facilities:       facJSON,
		ImageURL:         req.ImageURL,
		Status:           status,
		OperationalHours: opHours,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.roomRepo.Create(ctx, room); err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}

	s.logAudit(ctx, "room.created", room.ID, map[string]any{
		"name":        room.Name,
		"capacity":    room.Capacity,
		"category":    room.Category,
		"location":    room.Location,
		"status":      room.Status,
		"created_at":  room.CreatedAt,
	})

	return room, nil
}

func (s *RoomService) UpdateRoom(ctx context.Context, id string, req *domain.UpdateRoomRequest) (*domain.Room, error) {
	updated, err := s.roomRepo.Update(ctx, id, req)
	if err == nil && updated != nil {
		s.logAudit(ctx, "room.updated", id, req)
	}
	return updated, err
}

func (s *RoomService) DeleteRoom(ctx context.Context, id string) error {
	if err := s.roomRepo.Delete(ctx, id); err != nil {
		return err
	}
	s.logAudit(ctx, "room.deleted", id, map[string]string{"id": id})
	return nil
}
