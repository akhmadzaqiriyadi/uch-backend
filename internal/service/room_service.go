package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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

	return room, nil
}

func (s *RoomService) UpdateRoom(ctx context.Context, id string, req *domain.UpdateRoomRequest) (*domain.Room, error) {
	return s.roomRepo.Update(ctx, id, req)
}

func (s *RoomService) DeleteRoom(ctx context.Context, id string) error {
	return s.roomRepo.Delete(ctx, id)
}
