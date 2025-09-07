package event

import (
	"strings"
	"sync"
	"time"

	"github.com/kazegusuri/claude-companion/internal/logger"
	"github.com/kazegusuri/claude-companion/internal/narrator"
	"github.com/kazegusuri/claude-companion/internal/server/handler"
)

// Handler is the central event handler that processes all events
type Handler struct {
	sessionManager *handler.SessionManager
	narrator       narrator.Narrator
	printer        Printer
	emitter        handler.MessageEmitter
	eventChan      chan Event
	wg             sync.WaitGroup
	done           chan struct{}
	mu             sync.RWMutex
	subscribers    map[Type][]func(Event)
}

// NewHandler creates a new central event handler
func NewHandler(sessionManager *handler.SessionManager, narrator narrator.Narrator, printer Printer, emitter handler.MessageEmitter) *Handler {
	return &Handler{
		sessionManager: sessionManager,
		narrator:       narrator,
		printer:        printer,
		emitter:        emitter,
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
	case *SystemMessage:
		h.handleSystemMessage(e)
	case *SummaryEvent:
		h.handleSummaryEvent(e)
	case *EmitterEvent:
		h.handleEmitterEvent(e)
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
		// CWD is not available in Session anymore, use empty string
		h.sessionManager.CreateSession(event.Session.SessionID, "", "", event.Session.TranscriptPath)
	}

	// Generate narration based on event type
	h.generateNarration(event)

	// Print the notification
	if h.printer != nil {
		h.printer.Print(event)
	}
}

// generateNarration generates narration for notification events
func (h *Handler) generateNarration(event *NotificationEvent) {
	var narrationText string

	switch event.HookEventName {
	case "PreCompact":
		narrationText, _ = h.narrator.NarrateNotification(narrator.NotificationTypeCompact)
	case "SessionStart":
		// Determine notification type based on source
		var notificationType narrator.NotificationType
		switch event.Source {
		case "startup":
			notificationType = narrator.NotificationTypeSessionStartStartup
		case "clear":
			notificationType = narrator.NotificationTypeSessionStartClear
		case "resume":
			notificationType = narrator.NotificationTypeSessionStartResume
		default:
			notificationType = narrator.NotificationTypeSessionStartStartup
		}
		narrationText, _ = h.narrator.NarrateNotification(notificationType)
	case "Notification":
		// Check if it's a permission message
		isPermission, toolName := h.parsePermissionMessage(event.Message)
		if isPermission {
			narrationText, _ = h.narrator.NarrateToolUsePermission(toolName)
		} else {
			// General notification narration
			meta := &narrator.EventMeta{
				SessionID: event.Session.SessionID,
				CWD:       "", // CWD is not available in NotificationEvent
				Timestamp: time.Now(),
			}
			narrationText, _ = h.narrator.NarrateText(event.Message, false, meta)
		}
	}

	// Set narration if generated
	if narrationText != "" {
		event.Narration = &NarrationMessage{
			Text: narrationText,
		}
	}
}

// parsePermissionMessage checks if a message is a permission request and extracts tool name
func (h *Handler) parsePermissionMessage(message string) (isPermission bool, toolName string) {
	// Match "Claude will use <tool>: <operation>"
	if idx := strings.Index(message, "Claude will use "); idx == 0 {
		remaining := message[16:] // Skip "Claude will use "
		if colonIdx := strings.Index(remaining, ": "); colonIdx > 0 {
			toolNamePart := remaining[:colonIdx]
			// Check if it's an MCP tool
			if strings.HasPrefix(toolNamePart, "mcp__") {
				// Extract MCP server name and tool name
				parts := strings.SplitN(toolNamePart, "__", 3)
				if len(parts) >= 3 {
					toolName = toolNamePart // Return full MCP tool name
				} else {
					toolName = toolNamePart
				}
			} else {
				toolName = toolNamePart
			}
			return true, toolName
		}
	}
	return false, ""
}

// handleSystemMessage processes system message events
func (h *Handler) handleSystemMessage(event *SystemMessage) {
	// Send to WebSocket if not a meta message
	if h.emitter != nil && !event.SessionMessageBase.IsMeta {
		chatMsg := &handler.ChatMessage{
			Type:      handler.MessageTypeSystem,
			ID:        event.SessionMessageBase.UUID,
			Role:      handler.MessageRoleSystem,
			Text:      event.Content,
			Priority:  1,
			Timestamp: event.SessionMessageBase.Timestamp,
			Metadata: handler.Metadata{
				EventType: "system_message",
				SessionID: event.Session.SessionID,
				Role:      handler.MessageRoleSystem,
			},
		}
		h.emitter.BroadcastChat(chatMsg)
	}

	// Generate narration if needed
	if !event.SessionMessageBase.IsMeta || logger.IsDebugMode() {
		meta := &narrator.EventMeta{
			SessionID: event.Session.SessionID,
			CWD:       event.SessionMessageBase.CWD,
			Timestamp: event.SessionMessageBase.Timestamp,
		}
		narrationText, _ := h.narrator.NarrateText(event.Content, false, meta)
		if narrationText != "" {
			event.Narration = &NarrationMessage{
				Text: narrationText,
			}
		}
	}

	// Print the system message
	if h.printer != nil {
		h.printer.Print(event)
	}
}

// handleSummaryEvent processes summary events
func (h *Handler) handleSummaryEvent(event *SummaryEvent) {
	// Generate narration
	// Note: SummaryEvent doesn't have timestamp in the original JSON, so we use zero value
	meta := &narrator.EventMeta{
		SessionID: event.Session.SessionID,
		CWD:       "", // CWD is not available in SummaryEvent
		Timestamp: time.Time{}, // Summary events don't include timestamp
	}
	narrationText, _ := h.narrator.NarrateText(event.Summary, false, meta)
	if narrationText != "" {
		event.Narration = &NarrationMessage{
			Text: narrationText,
		}
	}

	// Print the summary
	if h.printer != nil {
		h.printer.Print(event)
	}
}

// handleEmitterEvent processes emitter events (WebSocket broadcasts)
func (h *Handler) handleEmitterEvent(event *EmitterEvent) {
	// Emit to WebSocket through emitter
	if h.emitter != nil {
		// Check if message is a ChatMessage
		if chatMsg, ok := event.Message.(*handler.ChatMessage); ok {
			h.emitter.BroadcastChat(chatMsg)
		}
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
