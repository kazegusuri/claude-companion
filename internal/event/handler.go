package event

import (
	"sync"

	"github.com/kazegusuri/claude-companion/internal/logger"
	"github.com/kazegusuri/claude-companion/internal/server/handler"
)

// Handler is the central event handler that processes all events
type Handler struct {
	sessionManager *handler.SessionManager
	eventChan      chan Event
	wg             sync.WaitGroup
	done           chan struct{}
	mu             sync.RWMutex
	subscribers    map[Type][]func(Event)
}

// NewHandler creates a new central event handler
func NewHandler(sessionManager *handler.SessionManager) *Handler {
	return &Handler{
		sessionManager: sessionManager,
		eventChan:      make(chan Event, 100),
		done:           make(chan struct{}),
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
	// Don't close eventChan here to prevent panic on send
	// The processEvents goroutine will exit when done is closed
	h.wg.Wait()
}

// SendEvent sends an event to be processed
func (h *Handler) SendEvent(event Event) {
	select {
	case <-h.done:
		// Handler is stopping, discard event
		return
	default:
		// Try to send event
		select {
		case h.eventChan <- event:
		case <-h.done:
			// Handler stopped while sending, discard event
		}
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
	default:
		logger.DebugWarning("Unknown event type in central handler: %T", event)
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

	logger.DebugInfo("Processed NotificationEvent: %s - %s", event.HookEventName, event.SessionID)
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
