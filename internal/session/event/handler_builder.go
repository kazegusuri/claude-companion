package event

import (
	"github.com/kazegusuri/claude-companion/internal/narrator"
	"github.com/kazegusuri/claude-companion/internal/server/handler"
)

// HandlerBuilder creates new Handler instances for each session
type HandlerBuilder struct {
	narrator       narrator.Narrator
	sessionManager *handler.SessionManager
	centralHandler CentralEventHandler
}

// NewHandlerBuilder creates a new handler builder
func NewHandlerBuilder(narrator narrator.Narrator, sessionManager *handler.SessionManager, centralHandler CentralEventHandler) *HandlerBuilder {
	return &HandlerBuilder{
		narrator:       narrator,
		sessionManager: sessionManager,
		centralHandler: centralHandler,
	}
}

// BuildFromPath creates a new Handler and Parser from a file path
func (b *HandlerBuilder) BuildFromPath(filePath string) (*Handler, *Parser) {
	// Create parser with the file path
	parser := NewParserWithPath(filePath)

	// Get session info from parser
	session := parser.GetSession()

	// Create handler with the session
	handler := NewHandler(b.narrator, b.sessionManager, b.centralHandler, session)

	return handler, parser
}
