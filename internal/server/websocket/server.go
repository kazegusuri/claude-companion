package websocket

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/kazegusuri/claude-companion/internal/server/handler"
)

// Client represents a WebSocket client connection
type Client struct {
	conn   *websocket.Conn
	send   chan *handler.ChatMessage
	id     string
	server *Server
}

// Server manages WebSocket connections and message broadcasting
type Server struct {
	clients       map[*Client]bool
	broadcast     chan *handler.ChatMessage
	register      chan *Client
	unregister    chan *Client
	mu            sync.RWMutex
	upgrader      websocket.Upgrader
	sessionGetter handler.SessionGetter
}

// NewServer creates a new WebSocket server
func NewServer(sessionGetter handler.SessionGetter) *Server {
	return &Server{
		clients:       make(map[*Client]bool),
		broadcast:     make(chan *handler.ChatMessage, 256),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		sessionGetter: sessionGetter,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// 開発環境では全てのオリジンを許可
				// 本番環境では適切な制限を設定してください
				return true
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
		send:   make(chan *handler.ChatMessage, 256),
		id:     uuid.New().String(),
		server: s,
	}

	s.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// BroadcastChat sends a chat message to all connected clients
func (s *Server) BroadcastChat(message *handler.ChatMessage) {
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

// clientMessageSender implements MessageSender interface for a specific client
type clientMessageSender struct {
	client *Client
	server *Server
}

// Send sends a message to the specific client
func (s *clientMessageSender) Send(msg *handler.ChatMessage) error {
	select {
	case s.client.send <- msg:
		return nil
	default:
		return fmt.Errorf("client %s send channel is full", s.client.id)
	}
}

// Broadcast sends a message to all connected clients
func (s *clientMessageSender) Broadcast(msg *handler.ChatMessage) {
	s.server.BroadcastChat(msg)
}

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

	// Create message sender for this client
	sender := &clientMessageSender{
		client: c,
		server: c.server,
	}

	// Create event handler for this client
	eventHandler := handler.NewEventHandler(c.id, sender, c.server.sessionGetter)

	for {
		var message handler.ClientMessage
		err := c.conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Delegate to event handler
		eventHandler.HandleMessage(message)
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
