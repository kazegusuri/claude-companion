package narrator

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kazegusuri/claude-companion/logger"
)

// VoiceNarrator wraps a narrator and adds voice output
type VoiceNarrator struct {
	narrator    Narrator
	voiceClient *VoiceVoxClient
	enabled     bool
	queue       *PriorityQueue
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	normalizer  *TextNormalizer
	translator  *CombinedTranslator
	metrics     *NarrationMetrics
}

// NewVoiceNarrator creates a new voice narrator
func NewVoiceNarrator(narrator Narrator, voiceClient *VoiceVoxClient, enabled bool) *VoiceNarrator {
	return NewVoiceNarratorWithTranslator(narrator, voiceClient, enabled, "", false)
}

// NewVoiceNarratorWithTranslator creates a new voice narrator with OpenAI translation support
func NewVoiceNarratorWithTranslator(narrator Narrator, voiceClient *VoiceVoxClient, enabled bool, openaiAPIKey string, useOpenAI bool) *VoiceNarrator {
	ctx, cancel := context.WithCancel(context.Background())

	vn := &VoiceNarrator{
		narrator:    narrator,
		voiceClient: voiceClient,
		enabled:     enabled,
		queue:       NewPriorityQueue(),
		ctx:         ctx,
		cancel:      cancel,
		normalizer:  NewTextNormalizer(),
		translator:  NewCombinedTranslator(openaiAPIKey, useOpenAI),
		metrics:     NewNarrationMetrics(),
	}

	if enabled && voiceClient != nil {
		// Check if VOICEVOX is available
		if !voiceClient.IsAvailable() {
			logger.LogWarning("VOICEVOX server is not available at %s", voiceClient.baseURL)
			vn.enabled = false
		} else {
			// Start voice worker
			vn.wg.Add(1)
			go vn.voiceWorker()
		}
	}

	return vn
}

// NarrateToolUse narrates tool usage with optional voice
func (vn *VoiceNarrator) NarrateToolUse(toolName string, input map[string]interface{}) string {
	text := vn.narrator.NarrateToolUse(toolName, input)

	if vn.enabled && text != "" {
		narType := NarrationTypeToolUse
		if isMCPTool(toolName) {
			narType = NarrationTypeToolUseMCP
		}

		vn.enqueueNarration(text, narType)
	}

	return text
}

// NarrateToolUsePermission narrates tool permission request with optional voice
func (vn *VoiceNarrator) NarrateToolUsePermission(toolName string) string {
	text := vn.narrator.NarrateToolUsePermission(toolName)

	if vn.enabled && text != "" {
		vn.enqueueNarration(text, NarrationTypeToolUsePermission)
	}

	return text
}

// NarrateText narrates text with optional voice
func (vn *VoiceNarrator) NarrateText(text string) string {
	result := vn.narrator.NarrateText(text)

	if vn.enabled && result != "" {
		vn.enqueueNarration(result, NarrationTypeText)
	}

	return result
}

// NarrateNotification narrates notification events with optional voice
func (vn *VoiceNarrator) NarrateNotification(notificationType NotificationType) string {
	text := vn.narrator.NarrateNotification(notificationType)

	if vn.enabled && text != "" {
		vn.enqueueNarration(text, NarrationTypeNotification)
	}

	return text
}

// NarrateTaskCompletion narrates task completion events with optional voice
func (vn *VoiceNarrator) NarrateTaskCompletion(description string, subagentType string) string {
	text := vn.narrator.NarrateTaskCompletion(description, subagentType)

	if vn.enabled && text != "" {
		vn.enqueueNarration(text, NarrationTypeNotification)
	}

	return text
}

// voiceWorker processes voice queue
func (vn *VoiceNarrator) voiceWorker() {
	defer vn.wg.Done()

	for {
		item := vn.queue.Dequeue(vn.ctx)
		if item == nil {
			return // context cancelled or queue closed
		}

		// Check if this item should be skipped
		if vn.queue.ShouldSkip(*item) {
			vn.metrics.IncrementSkipped()
			continue
		}

		// Create timeout context for each TTS operation
		ctx, cancel := context.WithTimeout(vn.ctx, 10*time.Second)

		// Try to synthesize and play
		if err := vn.voiceClient.TextToSpeech(ctx, item.Text); err != nil {
			vn.metrics.IncrementErrors()
			logger.LogError("Failed to play text to speech: %v", err)
		} else {
			vn.metrics.IncrementPlayed()
		}

		cancel()
	}
}

// SetEnabled enables or disables voice output
func (vn *VoiceNarrator) SetEnabled(enabled bool) {
	vn.enabled = enabled
}

// IsEnabled returns whether voice output is enabled
func (vn *VoiceNarrator) IsEnabled() bool {
	return vn.enabled
}

// Close stops the voice worker
func (vn *VoiceNarrator) Close() {
	vn.cancel()
	vn.queue.Close()
	vn.wg.Wait()
}

// Speak adds text to voice queue without returning it
func (vn *VoiceNarrator) Speak(text string) {
	if vn.enabled && text != "" {
		vn.enqueueNarration(text, NarrationTypeText)
	}
}

// enqueueNarration processes and enqueues a narration item
func (vn *VoiceNarrator) enqueueNarration(text string, narType NarrationType) {
	// Translate English to Japanese if needed
	ctx, cancel := context.WithTimeout(vn.ctx, 5*time.Second)
	translatedText, _ := vn.translator.Translate(ctx, text)
	cancel()

	// Normalize text for better TTS pronunciation
	normalizedText := vn.normalizer.Normalize(translatedText)

	item := NarrationItem{
		Text:      normalizedText,
		Type:      narType,
		Priority:  priorityMap[narType],
		Timestamp: time.Now(),
		ID:        uuid.New().String(),
	}

	if vn.queue.Enqueue(item) {
		vn.metrics.IncrementQueued()
	}
}

// isMCPTool checks if a tool name is an MCP tool
func isMCPTool(toolName string) bool {
	return strings.HasPrefix(toolName, "mcp__")
}

// GetMetrics returns current performance metrics
func (vn *VoiceNarrator) GetMetrics() map[string]interface{} {
	stats := vn.metrics.GetStats()
	stats["queue_size"] = vn.queue.Size()
	return stats
}
