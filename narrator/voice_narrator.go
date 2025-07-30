package narrator

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// VoiceNarrator wraps a narrator and adds voice output
type VoiceNarrator struct {
	narrator    Narrator
	voiceClient *VoiceVoxClient
	enabled     bool
	queue       chan string
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	normalizer  *TextNormalizer
	translator  *CombinedTranslator
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
		queue:       make(chan string, 100),
		ctx:         ctx,
		cancel:      cancel,
		normalizer:  NewTextNormalizer(),
		translator:  NewCombinedTranslator(openaiAPIKey, useOpenAI),
	}

	if enabled && voiceClient != nil {
		// Check if VOICEVOX is available
		if !voiceClient.IsAvailable() {
			fmt.Println("⚠️  Warning: VOICEVOX server is not available at", voiceClient.baseURL)
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
		// Translate English to Japanese if needed
		ctx, cancel := context.WithTimeout(vn.ctx, 5*time.Second)
		translatedText, _ := vn.translator.Translate(ctx, text)
		cancel()

		// Normalize text for better TTS pronunciation
		normalizedText := vn.normalizer.Normalize(translatedText)
		select {
		case vn.queue <- normalizedText:
		default:
			// Queue is full, skip voice output
		}
	}

	return text
}

// NarrateToolUsePermission narrates tool permission request with optional voice
func (vn *VoiceNarrator) NarrateToolUsePermission(toolName string) string {
	text := vn.narrator.NarrateToolUsePermission(toolName)

	if vn.enabled && text != "" {
		// Translate English to Japanese if needed
		ctx, cancel := context.WithTimeout(vn.ctx, 5*time.Second)
		translatedText, _ := vn.translator.Translate(ctx, text)
		cancel()

		// Normalize text for better TTS pronunciation
		normalizedText := vn.normalizer.Normalize(translatedText)
		select {
		case vn.queue <- normalizedText:
		default:
			// Queue is full, skip voice output
		}
	}

	return text
}

// NarrateText narrates text with optional voice
func (vn *VoiceNarrator) NarrateText(text string) string {
	result := vn.narrator.NarrateText(text)

	if vn.enabled && result != "" {
		// Translate English to Japanese if needed
		ctx, cancel := context.WithTimeout(vn.ctx, 5*time.Second)
		translatedText, _ := vn.translator.Translate(ctx, result)
		cancel()

		// Normalize text for better TTS pronunciation
		normalizedText := vn.normalizer.Normalize(translatedText)
		select {
		case vn.queue <- normalizedText:
		default:
			// Queue is full, skip voice output
		}
	}

	return result
}

// NarrateNotification narrates notification events with optional voice
func (vn *VoiceNarrator) NarrateNotification(notificationType NotificationType) string {
	text := vn.narrator.NarrateNotification(notificationType)

	if vn.enabled && text != "" {
		// Translate English to Japanese if needed
		ctx, cancel := context.WithTimeout(vn.ctx, 5*time.Second)
		translatedText, _ := vn.translator.Translate(ctx, text)
		cancel()

		// Normalize text for better TTS pronunciation
		normalizedText := vn.normalizer.Normalize(translatedText)
		select {
		case vn.queue <- normalizedText:
		default:
			// Queue is full, skip voice output
		}
	}

	return text
}

// voiceWorker processes voice queue
func (vn *VoiceNarrator) voiceWorker() {
	defer vn.wg.Done()

	for {
		select {
		case text := <-vn.queue:
			// Create timeout context for each TTS operation
			ctx, cancel := context.WithTimeout(vn.ctx, 10*time.Second)

			// Try to synthesize and play
			if err := vn.voiceClient.TextToSpeech(ctx, text); err != nil {
				// Silently ignore errors to not interrupt the main flow
				// Could log to debug if needed
			}

			cancel()

		case <-vn.ctx.Done():
			return
		}
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
	close(vn.queue)
	vn.wg.Wait()
}

// Speak adds text to voice queue without returning it
func (vn *VoiceNarrator) Speak(text string) {
	if vn.enabled && text != "" {
		// Translate English to Japanese if needed
		ctx, cancel := context.WithTimeout(vn.ctx, 5*time.Second)
		translatedText, _ := vn.translator.Translate(ctx, text)
		cancel()

		// Normalize text for better TTS pronunciation
		normalizedText := vn.normalizer.Normalize(translatedText)
		select {
		case vn.queue <- normalizedText:
		default:
			// Queue is full, skip voice output
		}
	}
}
