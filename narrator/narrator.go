package narrator

import (
	"strings"
)

// NotificationType represents different types of notifications
type NotificationType string

const (
	NotificationTypeCompact             NotificationType = "compact"
	NotificationTypeSessionStartStartup NotificationType = "session_start_startup"
	NotificationTypeSessionStartClear   NotificationType = "session_start_clear"
	NotificationTypeSessionStartResume  NotificationType = "session_start_resume"
	NotificationTypeSessionStartCompact NotificationType = "session_start_compact"
)

// Narrator interface for converting tool actions to natural language
type Narrator interface {
	NarrateToolUse(toolName string, input map[string]interface{}) (string, bool)
	NarrateToolUsePermission(toolName string) (string, bool)
	NarrateText(text string) (string, bool)
	NarrateNotification(notificationType NotificationType) (string, bool)
	NarrateTaskCompletion(description string, subagentType string) (string, bool)
}

// Helper function to extract domain from URL
func extractDomain(url string) string {
	// Simple domain extraction
	if strings.HasPrefix(url, "http://") {
		url = strings.TrimPrefix(url, "http://")
	} else if strings.HasPrefix(url, "https://") {
		url = strings.TrimPrefix(url, "https://")
	}

	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		domain := parts[0]
		// Remove port if present
		if colonIdx := strings.Index(domain, ":"); colonIdx != -1 {
			domain = domain[:colonIdx]
		}
		return domain
	}
	return url
}

// NoOpNarrator is a narrator that returns empty strings for all operations
type NoOpNarrator struct{}

// NewNoOpNarrator creates a new no-op narrator
func NewNoOpNarrator() *NoOpNarrator {
	return &NoOpNarrator{}
}

// NarrateToolUse returns empty string
func (n *NoOpNarrator) NarrateToolUse(toolName string, input map[string]interface{}) (string, bool) {
	return "", true
}

// NarrateToolUsePermission returns empty string
func (n *NoOpNarrator) NarrateToolUsePermission(toolName string) (string, bool) {
	return "", true
}

// NarrateText returns the text as-is
func (n *NoOpNarrator) NarrateText(text string) (string, bool) {
	return text, false
}

// NarrateNotification returns empty string
func (n *NoOpNarrator) NarrateNotification(notificationType NotificationType) (string, bool) {
	return "", true
}

// NarrateTaskCompletion returns empty string
func (n *NoOpNarrator) NarrateTaskCompletion(description string, subagentType string) (string, bool) {
	return "", true
}
