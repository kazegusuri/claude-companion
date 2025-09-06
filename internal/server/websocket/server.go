package websocket

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/kazegusuri/claude-companion/internal/server/db"
	"github.com/kazegusuri/claude-companion/internal/server/handler"
)

// Client represents a WebSocket client connection
type Client struct {
	conn             *websocket.Conn
	send             chan *handler.ChatMessage
	id               string
	server           *Server
	state            *handler.WebSocketConnectionState // Current client state
	currentSessionID string                            // Current session ID being monitored (for agent mode)
	stopWatcher      chan bool                         // Channel to stop the session watcher goroutine
	mu               sync.Mutex                        // Protects state and currentSessionID
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
	database      *db.DB // Database for agent session lookups
}

// NewServer creates a new WebSocket server
func NewServer(sessionGetter handler.SessionGetter, database *db.DB) *Server {
	return &Server{
		clients:       make(map[*Client]bool),
		broadcast:     make(chan *handler.ChatMessage, 256),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		sessionGetter: sessionGetter,
		database:      database,
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

		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.send)
				s.mu.Unlock()
			} else {
				s.mu.Unlock()
			}

		case message := <-s.broadcast:
			s.mu.RLock()
			for client := range s.clients {
				// Filter messages for agent mode clients
				client.mu.Lock()
				state := client.state
				currentSessionID := client.currentSessionID
				client.mu.Unlock()

				if state != nil && state.Mode == "agent" && state.AgentPID != nil {
					// Only send messages from the current session
					if message.Metadata.SessionID != currentSessionID {
						continue
					}
				}

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
		conn:        conn,
		send:        make(chan *handler.ChatMessage, 256),
		id:          uuid.New().String(),
		server:      s,
		state:       &handler.WebSocketConnectionState{Mode: "timeline"},
		stopWatcher: make(chan bool),
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

	// WORKAROUND: Resolve agent ID from session ID here
	// TODO: Move this logic to event/handler for better separation of concerns
	if s.database != nil && message.Metadata.SessionID != "" {
		agentPID, err := s.database.GetAgentPIDBySessionID(message.Metadata.SessionID)
		if err == nil && agentPID != nil {
			message.Metadata.AgentID = agentPID
		}
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

// UpdateState updates the client's WebSocket connection state
func (c *Client) UpdateState(newState *handler.WebSocketConnectionState) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Stop existing watcher if switching from agent mode
	if c.state != nil && c.state.Mode == "agent" && c.state.AgentPID != nil {
		select {
		case c.stopWatcher <- true:
		default:
		}
	}

	// Update state
	c.state = newState

	// Handle mode-specific logic
	if newState.Mode == "agent" && newState.AgentPID != nil {
		c.currentSessionID = ""
		// Start new watcher for agent mode
		if c.server.database != nil {
			go c.watchAgentSession()
		}
	} else {
		c.currentSessionID = ""
	}
}

// watchAgentSession monitors the agent's session changes and updates the filter
func (c *Client) watchAgentSession() {
	// Function to update session ID
	updateSession := func() {
		c.mu.Lock()
		state := c.state
		c.mu.Unlock()

		if state == nil || state.Mode != "agent" || state.AgentPID == nil || c.server.database == nil {
			return
		}

		// Get current session for this agent
		agent, err := c.server.database.GetClaudeAgent(*state.AgentPID)
		if err != nil {
			log.Printf("Error getting agent %d: %v", *state.AgentPID, err)
			return
		}

		if agent == nil {
			// Agent no longer exists, might want to disconnect
			return
		}

		// Update session ID if changed
		c.mu.Lock()
		if agent.SessionID != c.currentSessionID {
			c.currentSessionID = agent.SessionID
		}
		c.mu.Unlock()
	}

	// Get initial session ID immediately
	updateSession()

	// Then check periodically
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			updateSession()

		case <-c.stopWatcher:
			return
		}
	}
}

// readPump pumps messages from the websocket connection to the server
func (c *Client) readPump() {
	defer func() {
		// Stop the session watcher if running
		c.mu.Lock()
		if c.state != nil && c.state.Mode == "agent" && c.state.AgentPID != nil {
			close(c.stopWatcher)
		}
		c.mu.Unlock()
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

		// Handle state update messages directly
		switch message.Type {
		case handler.MessageTypeUpdateState:
			if message.State != nil {
				c.UpdateState(message.State)
				// No confirmation message needed
			}
		default:
			// Delegate to event handler for other message types
			eventHandler.HandleMessage(message)
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
