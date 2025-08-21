package speech

import (
	"encoding/base64"
	"time"

	"github.com/google/uuid"
	"github.com/kazegusuri/claude-companion/handler"
	"github.com/kazegusuri/claude-companion/websocket"
)

// WebSocketPlayer implements Player interface for WebSocket audio streaming
type WebSocketPlayer struct {
	server *websocket.Server
}

// NewWebSocketPlayer creates a new WebSocket player
func NewWebSocketPlayer(server *websocket.Server) *WebSocketPlayer {
	return &WebSocketPlayer{
		server: server,
	}
}

// Play sends audio data and metadata via WebSocket
func (p *WebSocketPlayer) Play(audioData []byte, meta *AudioMeta) error {
	// Encode audio data to base64
	audioDataBase64 := base64.StdEncoding.EncodeToString(audioData)

	// Create metadata
	metadata := handler.Metadata{
		EventType:  "audio",
		SampleRate: 24000, // Default sample rate (matching common VOICEVOX output)
	}

	// Add duration if available
	if meta != nil && meta.Duration > 0 {
		metadata.Duration = meta.Duration.Seconds()
	}

	// Create the message
	message := &handler.AudioMessage{
		Type:      handler.MessageTypeAudio,
		ID:        uuid.New().String(),
		Text:      "",
		AudioData: audioDataBase64,
		Priority:  5,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	// Add text if available
	if meta != nil {
		if meta.NormalizedText != "" {
			message.Text = meta.NormalizedText
		} else if meta.OriginalText != "" {
			message.Text = meta.OriginalText
		}
	}

	// Broadcast the message to all connected clients
	p.server.Broadcast(message)

	return nil
}

// TestPlay sends a test message with silent WAV data
func (p *WebSocketPlayer) TestPlay() error {
	// Get silent WAV data
	silentData := GetSilentWAV()

	// Create test metadata
	meta := &AudioMeta{
		OriginalText:   "Test audio",
		NormalizedText: "Test audio",
		Duration:       50 * time.Millisecond, // Very short duration for test
	}

	// Send via Play method
	return p.Play(silentData, meta)
}
