package domain

import (
	"context"
	"encoding/json"
	"time"
)

type Room struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Slug             string          `json:"slug"`
	Category         string          `json:"category"`
	Capacity         int             `json:"capacity"`
	Location         string          `json:"location"`
	Description      string          `json:"description"`
	Facilities       json.RawMessage `json:"facilities"`
	ImageURL         string          `json:"image_url"`
	Status           string          `json:"status"` // available, maintenance, reserved
	OperationalHours string          `json:"operational_hours"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type CreateRoomRequest struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Slug             string   `json:"slug"`
	Category         string   `json:"category"`
	Capacity         int      `json:"capacity"`
	Location         string   `json:"location"`
	Description      string   `json:"description"`
	Facilities       []string `json:"facilities"`
	ImageURL         string   `json:"image_url"`
	Status           string   `json:"status"`
	OperationalHours string   `json:"operational_hours"`
}

type UpdateRoomRequest struct {
	Name             *string   `json:"name"`
	Slug             *string   `json:"slug"`
	Category         *string   `json:"category"`
	Capacity         *int      `json:"capacity"`
	Location         *string   `json:"location"`
	Description      *string   `json:"description"`
	Facilities       *[]string `json:"facilities"`
	ImageURL         *string   `json:"image_url"`
	Status           *string   `json:"status"`
	OperationalHours *string   `json:"operational_hours"`
}

type RoomRepository interface {
	Create(ctx context.Context, room *Room) error
	GetByID(ctx context.Context, id string) (*Room, error)
	List(ctx context.Context) ([]Room, error)
	Update(ctx context.Context, id string, req *UpdateRoomRequest) (*Room, error)
	Delete(ctx context.Context, id string) error
}

