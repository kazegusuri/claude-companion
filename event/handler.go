package event

import (
	"fmt"
	"log"
	"sync"

	"github.com/kazegusuri/claude-companion/narrator"
)

// Handler processes events from multiple sources
type Handler struct {
	narrator  narrator.Narrator
	formatter *Formatter
	debugMode bool
	eventChan chan Event
	wg        sync.WaitGroup
	done      chan struct{}
}

// NewHandler creates a new event handler
func NewHandler(narrator narrator.Narrator, debugMode bool) *Handler {
	formatter := NewFormatter(narrator)
	formatter.SetDebugMode(debugMode)

	return &Handler{
		narrator:  narrator,
		formatter: formatter,
		debugMode: debugMode,
		eventChan: make(chan Event, 100),
		done:      make(chan struct{}),
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
	switch e := event.(type) {
	case *NotificationEvent:
		// Process notification events
		output, err := h.formatter.Format(e)
		if err != nil {
			if h.debugMode {
				log.Printf("Error formatting NotificationEvent: %v", err)
			}
			return
		}
		if output != "" {
			fmt.Print(output)
		}
	case *UserMessage, *AssistantMessage, *SystemMessage, *SummaryEvent, *BaseEvent:
		// Format and display parsed events
		output, err := h.formatter.Format(e)
		if err != nil {
			if h.debugMode {
				log.Printf("Error formatting %T: %v", e, err)
			}
			return
		}
		if output != "" {
			fmt.Print(output)
		}
	default:
		if h.debugMode {
			log.Printf("Unknown event type: %T", event)
		}
	}
}
