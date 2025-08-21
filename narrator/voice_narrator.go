package narrator

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kazegusuri/claude-companion/logger"
	"github.com/kazegusuri/claude-companion/speech"
)

// VoiceNarrator wraps a narrator and adds voice output
type VoiceNarrator struct {
	narrator    Narrator
	synthesizer speech.Synthesizer
	player      speech.Player
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
func NewVoiceNarrator(narrator Narrator, synthesizer speech.Synthesizer, player speech.Player, enabled bool) *VoiceNarrator {
	return NewVoiceNarratorWithTranslator(narrator, synthesizer, player, enabled, "", false)
}

// NewVoiceNarratorWithTranslator creates a new voice narrator with OpenAI translation support
func NewVoiceNarratorWithTranslator(narrator Narrator, synthesizer speech.Synthesizer, player speech.Player, enabled bool, openaiAPIKey string, useOpenAI bool) *VoiceNarrator {
	ctx, cancel := context.WithCancel(context.Background())

	vn := &VoiceNarrator{
		narrator:    narrator,
		synthesizer: synthesizer,
		player:      player,
		enabled:     enabled,
		queue:       NewPriorityQueue(),
		ctx:         ctx,
		cancel:      cancel,
		normalizer:  NewTextNormalizer(),
		translator:  NewCombinedTranslator(openaiAPIKey, useOpenAI),
		metrics:     NewNarrationMetrics(),
	}

	if enabled && synthesizer != nil && player != nil {
		// Check if synthesizer is available
		if !synthesizer.IsAvailable() {
			logger.LogWarning("Speech synthesizer is not available")
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
func (vn *VoiceNarrator) NarrateToolUse(toolName string, input map[string]interface{}) (string, bool) {
	text, shouldFallback := vn.narrator.NarrateToolUse(toolName, input)

	if vn.enabled && text != "" {
		narType := NarrationTypeToolUse
		if isMCPTool(toolName) {
			narType = NarrationTypeToolUseMCP
		}

		vn.enqueueNarration(text, narType, nil)
	}

	return text, shouldFallback
}

// NarrateToolUsePermission narrates tool permission request with optional voice
func (vn *VoiceNarrator) NarrateToolUsePermission(toolName string) (string, bool) {
	text, shouldFallback := vn.narrator.NarrateToolUsePermission(toolName)

	if vn.enabled && text != "" {
		vn.enqueueNarration(text, NarrationTypeToolUsePermission, nil)
	}

	return text, shouldFallback
}

// NarrateText narrates text with optional voice
func (vn *VoiceNarrator) NarrateText(text string, isThinking bool, meta *EventMeta) (string, bool) {
	result, shouldFallback := vn.narrator.NarrateText(text, isThinking, meta)

	if vn.enabled && result != "" {
		vn.enqueueNarration(result, NarrationTypeText, meta)
	}

	return result, shouldFallback
}

// NarrateNotification narrates notification events with optional voice
func (vn *VoiceNarrator) NarrateNotification(notificationType NotificationType) (string, bool) {
	text, shouldFallback := vn.narrator.NarrateNotification(notificationType)

	if vn.enabled && text != "" {
		vn.enqueueNarration(text, NarrationTypeNotification, nil)
	}

	return text, shouldFallback
}

// NarrateTaskCompletion narrates task completion events with optional voice
func (vn *VoiceNarrator) NarrateTaskCompletion(description string, subagentType string) (string, bool) {
	text, shouldFallback := vn.narrator.NarrateTaskCompletion(description, subagentType)

	if vn.enabled && text != "" {
		vn.enqueueNarration(text, NarrationTypeNotification, nil)
	}

	return text, shouldFallback
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
		ctx, cancel := context.WithTimeout(vn.ctx, 15*time.Second)

		// Try to synthesize
		audioData, err := vn.synthesizer.Synthesize(ctx, item.Text)
		cancel()

		if err != nil {
			vn.metrics.IncrementErrors()
			logger.LogError("Failed to synthesize speech: %v", err)
			continue
		}

		// Create audio metadata
		meta := &speech.AudioMeta{
			OriginalText:   item.OriginalText,
			NormalizedText: item.Text,
		}

		// Add SessionID from EventMeta if available
		if item.Meta != nil && item.Meta.SessionID != "" {
			meta.SessionID = item.Meta.SessionID
		}

		// Parse audio duration
		if duration, err := speech.ParseWAVDuration(audioData); err == nil {
			meta.Duration = duration
		} else {
			// Log error but continue processing
			logger.LogWarning("Failed to parse WAV duration: %v", err)
		}

		// Play audio with metadata
		if err := vn.player.Play(audioData, meta); err != nil {
			vn.metrics.IncrementErrors()
			logger.LogError("Failed to play audio: %v", err)
		} else {
			vn.metrics.IncrementPlayed()
		}
	}
}

// Close stops the voice worker
func (vn *VoiceNarrator) Close() {
	vn.cancel()
	vn.queue.Close()
	vn.wg.Wait()
}

// enqueueNarration processes and enqueues a narration item
func (vn *VoiceNarrator) enqueueNarration(text string, narType NarrationType, meta *EventMeta) {
	// Translate English to Japanese if needed
	ctx, cancel := context.WithTimeout(vn.ctx, 5*time.Second)
	translatedText, _ := vn.translator.Translate(ctx, text)
	cancel()

	// Normalize text for better TTS pronunciation
	normalizedText := vn.normalizer.Normalize(translatedText)

	item := NarrationItem{
		Text:         normalizedText,
		OriginalText: translatedText, // Use translated text as original
		Type:         narType,
		Priority:     priorityMap[narType],
		Timestamp:    time.Now(),
		ID:           uuid.New().String(),
		Meta:         meta,
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
