package realtime

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all frontend origins in development and campus domain
	},
}

type Message struct {
	Event     string    `json:"event"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

type Client struct {
	Hub    *Hub
	Conn   *websocket.Conn
	Send   chan []byte
	UserID uuid.UUID
	Role   string
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			total := len(h.clients)
			h.mu.Unlock()
			slog.Info("Realtime client connected", slog.String("user_id", client.UserID.String()), slog.String("role", client.Role), slog.Int("total_clients", total))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			total := len(h.clients)
			h.mu.Unlock()
			slog.Info("Realtime client disconnected", slog.String("user_id", client.UserID.String()), slog.Int("total_clients", total))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Broadcast(event string, payload any) {
	msg := Message{
		Event:     event,
		Timestamp: time.Now().UTC(),
		Data:      payload,
	}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	total := len(h.clients)
	h.mu.RUnlock()

	slog.Info("Realtime Broadcast event", slog.String("event", event), slog.Int("total_clients", total))
	h.broadcast <- bytes
}

func (h *Hub) SendToUser(userID uuid.UUID, event string, payload any) {
	msg := Message{
		Event:     event,
		Timestamp: time.Now().UTC(),
		Data:      payload,
	}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	matched := 0
	for client := range h.clients {
		if client.UserID == userID {
			select {
			case client.Send <- bytes:
				matched++
			default:
			}
		}
	}
	slog.Info("Realtime SendToUser", slog.String("user_id", userID.String()), slog.String("event", event), slog.Int("matched_clients", matched), slog.Int("total_clients", len(h.clients)))
}

func (h *Hub) SendToRole(role string, event string, payload any) {
	msg := Message{
		Event:     event,
		Timestamp: time.Now().UTC(),
		Data:      payload,
	}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	matched := 0
	for client := range h.clients {
		if client.Role == role || client.Role == "admin" {
			select {
			case client.Send <- bytes:
				matched++
			default:
			}
		}
	}
	slog.Info("Realtime SendToRole", slog.String("role", role), slog.String("event", event), slog.Int("matched_clients", matched), slog.Int("total_clients", len(h.clients)))
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		_ = c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request, userID uuid.UUID, role string) error {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	client := &Client{
		Hub:    hub,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: userID,
		Role:   role,
	}

	client.Hub.register <- client

	go client.WritePump()
	go client.ReadPump()

	return nil
}
