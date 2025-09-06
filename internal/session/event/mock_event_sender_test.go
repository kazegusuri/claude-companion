package event

import (
	"sync"
)

// MockEventSender implements EventSender for testing
type MockEventSender struct {
	events []Event
	mu     sync.Mutex
}

func NewMockEventSender() *MockEventSender {
	return &MockEventSender{
		events: make([]Event, 0),
	}
}

func (m *MockEventSender) SendEvent(event Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func (m *MockEventSender) GetEvents() []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Event{}, m.events...)
}

func (m *MockEventSender) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = make([]Event, 0)
}

// Ensure MockEventSender implements EventSender
var _ EventSender = (*MockEventSender)(nil)
