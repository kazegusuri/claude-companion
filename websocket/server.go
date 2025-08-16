package websocket

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	MessageTypeAudio MessageType = "audio"
	MessageTypeText  MessageType = "text"
	MessageTypePing  MessageType = "ping"
	MessageTypePong  MessageType = "pong"
	MessageTypeError MessageType = "error"
)

// AudioMessage represents a message containing audio data
type AudioMessage struct {
	Type      MessageType `json:"type"`
	ID        string      `json:"id"`
	Text      string      `json:"text"`
	AudioData string      `json:"audioData,omitempty"` // Base64 encoded WAV data
	Priority  int         `json:"priority"`
	Timestamp time.Time   `json:"timestamp"`
	Metadata  Metadata    `json:"metadata,omitempty"`
}

// Metadata contains additional information about the message
type Metadata struct {
	EventType  string  `json:"eventType"`
	ToolName   string  `json:"toolName,omitempty"`
	Speaker    int     `json:"speaker,omitempty"`
	SampleRate int     `json:"sampleRate,omitempty"`
	Duration   float64 `json:"duration,omitempty"`
}

// Client represents a WebSocket client connection
type Client struct {
	conn   *websocket.Conn
	send   chan *AudioMessage
	id     string
	server *Server
}

// Server manages WebSocket connections and message broadcasting
type Server struct {
	clients    map[*Client]bool
	broadcast  chan *AudioMessage
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	upgrader   websocket.Upgrader
}

// NewServer creates a new WebSocket server
func NewServer() *Server {
	return &Server{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *AudioMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from localhost in development
				origin := r.Header.Get("Origin")
				return origin == "http://localhost:3000" || origin == "http://localhost:3001"
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// Run starts the server's main loop
func (s *Server) Run() {
	for {
		select {
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client] = true
			s.mu.Unlock()
			log.Printf("Client connected: %s", client.id)

		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.send)
				s.mu.Unlock()
				log.Printf("Client disconnected: %s", client.id)
			} else {
				s.mu.Unlock()
			}

		case message := <-s.broadcast:
			s.mu.RLock()
			for client := range s.clients {
				select {
				case client.send <- message:
				default:
					// Client's send channel is full, close it
					close(client.send)
					delete(s.clients, client)
				}
			}
			s.mu.RUnlock()
		}
	}
}

// HandleWebSocket handles WebSocket upgrade requests
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		conn:   conn,
		send:   make(chan *AudioMessage, 256),
		id:     uuid.New().String(),
		server: s,
	}

	s.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// Broadcast sends a message to all connected clients
func (s *Server) Broadcast(message *AudioMessage) {
	if message.ID == "" {
		message.ID = uuid.New().String()
	}
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}
	s.broadcast <- message
}

// GetClientCount returns the number of connected clients
func (s *Server) GetClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512KB
)

// readPump pumps messages from the websocket connection to the server
func (c *Client) readPump() {
	defer func() {
		c.server.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var message AudioMessage
		err := c.conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle ping messages
		if message.Type == MessageTypePing {
			pong := &AudioMessage{
				Type:      MessageTypePong,
				ID:        uuid.New().String(),
				Timestamp: time.Now(),
			}
			select {
			case c.send <- pong:
			default:
				// Channel full, skip
			}
		}
	}
}

// writePump pumps messages from the server to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The server closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
