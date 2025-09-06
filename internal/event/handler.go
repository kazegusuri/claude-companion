package event

import (
	"sync"

	"github.com/kazegusuri/claude-companion/internal/logger"
	"github.com/kazegusuri/claude-companion/internal/server/handler"
)

// EventProcessor is an interface for processing events
type EventProcessor interface {
	SendEventInterface(event interface{})
}

// Handler is the central event handler that processes all events
type Handler struct {
	sessionManager   *handler.SessionManager
	sessionProcessor EventProcessor
	eventChan        chan Event
	wg               sync.WaitGroup
	done             chan struct{}
	debugMode        bool
	mu               sync.RWMutex
	subscribers      map[Type][]func(Event)
}

// NewHandler creates a new central event handler
func NewHandler(sessionManager *handler.SessionManager, debugMode bool) *Handler {
	return &Handler{
		sessionManager: sessionManager,
		eventChan:      make(chan Event, 100),
		done:           make(chan struct{}),
		debugMode:      debugMode,
		subscribers:    make(map[Type][]func(Event)),
	}
}

// Start begins processing events
func (h *Handler) Start() {
	h.wg.Add(1)
	go h.processEvents()
}

// Stop stops the event handler
func (h *Handler) Stop() {
	close(h.done)
	close(h.eventChan)
	h.wg.Wait()
}

// SendEvent sends an event to be processed
func (h *Handler) SendEvent(event Event) {
	select {
	case h.eventChan <- event:
	case <-h.done:
		// Handler is stopping, discard event
	}
}

// Subscribe adds a subscriber for a specific event type
func (h *Handler) Subscribe(eventType Type, handler func(Event)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.subscribers[eventType] = append(h.subscribers[eventType], handler)
}

// processEvents processes events from the channel
func (h *Handler) processEvents() {
	defer h.wg.Done()

	for {
		select {
		case event, ok := <-h.eventChan:
			if !ok {
				return
			}
			h.processEvent(event)
		case <-h.done:
			// Drain remaining events
			for {
				select {
				case event, ok := <-h.eventChan:
					if !ok {
						return
					}
					h.processEvent(event)
				default:
					return
				}
			}
		}
	}
}

// processEvent processes a single event based on its type
func (h *Handler) processEvent(event Event) {
	eventType := event.Type()

	// Handle specific event types
	switch e := event.(type) {
	case *NotificationEvent:
		h.handleNotificationEvent(e)
	case *SessionEvent:
		h.handleSessionEvent(e)
	case *FrontendEvent:
		h.handleFrontendEvent(e)
	default:
		if h.debugMode {
			logger.LogWarning("Unknown event type in central handler: %T", event)
		}
	}

	// Notify subscribers
	h.notifySubscribers(eventType, event)
}

// handleNotificationEvent processes notification events
func (h *Handler) handleNotificationEvent(event *NotificationEvent) {
	// Create session through SessionManager for SessionStart events
	if event.HookEventName == "SessionStart" {
		h.sessionManager.CreateSession(event.SessionID, "", event.CWD, event.TranscriptPath)
	}

	// Forward to session processor if available
	if h.sessionProcessor != nil {
		h.sessionProcessor.SendEventInterface(event)
	}

	if h.debugMode {
		logger.LogInfo("Processed NotificationEvent: %s - %s", event.HookEventName, event.SessionID)
	}
}

// handleSessionEvent processes session-related events
func (h *Handler) handleSessionEvent(event *SessionEvent) {
	switch event.EventType {
	case "create":
		h.sessionManager.CreateSession(event.SessionID, event.UUID, event.CWD, event.TranscriptPath)
	case "update":
		// Handle session updates if needed
		if h.debugMode {
			logger.LogInfo("Session update event: %s", event.SessionID)
		}
	case "delete":
		// Handle session deletion if needed
		if h.debugMode {
			logger.LogInfo("Session delete event: %s", event.SessionID)
		}
	}
}

// handleFrontendEvent processes events from the frontend
func (h *Handler) handleFrontendEvent(event *FrontendEvent) {
	// Process frontend events based on their type
	if h.debugMode {
		logger.LogInfo("Frontend event: %s", event.EventType)
	}

	// Example: Handle specific frontend event types
	switch event.EventType {
	case "session_request":
		// Handle session request from frontend
	case "command":
		// Handle command from frontend
	default:
		// Handle other frontend events
	}
}

// notifySubscribers notifies all subscribers of an event type
func (h *Handler) notifySubscribers(eventType Type, event Event) {
	h.mu.RLock()
	subscribers := h.subscribers[eventType]
	h.mu.RUnlock()

	for _, subscriber := range subscribers {
		// Call subscriber in a goroutine to avoid blocking
		go func(handler func(Event)) {
			defer func() {
				if r := recover(); r != nil {
					logger.LogError("Subscriber panic: %v", r)
				}
			}()
			handler(event)
		}(subscriber)
	}
}

// GetSessionManager returns the session manager
func (h *Handler) GetSessionManager() *handler.SessionManager {
	return h.sessionManager
}

// SetSessionProcessor sets the session event processor
func (h *Handler) SetSessionProcessor(processor EventProcessor) {
	h.sessionProcessor = processor
}
