package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/kazegusuri/claude-companion/internal/server/handler"
	"github.com/kazegusuri/claude-companion/internal/server/websocket"
	"github.com/kazegusuri/claude-companion/internal/speech"
	"github.com/rs/cors"
)

func main() {
	// Create WebSocket server (nil SessionGetter for test)
	wsServer := websocket.NewServer(nil)
	go wsServer.Run()

	// Create WebSocket player for audio messages
	wsPlayer := speech.NewWebSocketPlayer(wsServer)

	// HTTP handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/audio", wsServer.HandleWebSocket)
	mux.HandleFunc("/health", handleHealth(wsServer))
	mux.HandleFunc("/api/send/text", handleSendText(wsServer))
	mux.HandleFunc("/api/send/audio", handleSendAudio(wsPlayer))
	mux.HandleFunc("/api/send/test", handleSendTest(wsPlayer))

	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(mux)

	port := ":8080"
	log.Printf("WebSocket test server starting on %s", port)
	log.Println("Endpoints:")
	log.Println("  WebSocket: ws://localhost:8080/ws/audio")
	log.Println("  Health:    http://localhost:8080/health")
	log.Println("  Send Text: POST http://localhost:8080/api/send/text")
	log.Println("  Send Audio: POST http://localhost:8080/api/send/audio")
	log.Println("  Send Test: POST http://localhost:8080/api/send/test")
	log.Fatal(http.ListenAndServe(port, handler))
}

func handleHealth(server *websocket.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","clients":%d}`, server.GetClientCount())
	}
}

// TextRequest represents a request to send text
type TextRequest struct {
	Text      string `json:"text"`
	EventType string `json:"eventType,omitempty"`
	ToolName  string `json:"toolName,omitempty"`
}

func handleSendText(server *websocket.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req TextRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Create message with assistant type
		msg := &handler.ChatMessage{
			Type:      handler.MessageTypeAssistant,
			Role:      handler.MessageRoleAssistant,
			SubType:   handler.AssistantMessageSubTypeText,
			ID:        uuid.New().String(),
			Text:      req.Text,
			Priority:  5,
			Timestamp: time.Now(),
		}

		// Add metadata if provided
		if req.EventType != "" || req.ToolName != "" {
			metadata := handler.Metadata{}
			if req.EventType != "" {
				metadata.EventType = req.EventType
			}
			if req.ToolName != "" {
				metadata.ToolName = req.ToolName
			}
			msg.Metadata = metadata
		}

		// Broadcast message
		server.BroadcastChat(msg)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"id":      msg.ID,
		})
	}
}

// AudioRequest represents a request to send audio
type AudioRequest struct {
	Text       string  `json:"text"`
	AudioData  string  `json:"audioData,omitempty"` // Base64 encoded
	EventType  string  `json:"eventType,omitempty"`
	ToolName   string  `json:"toolName,omitempty"`
	SampleRate int     `json:"sampleRate,omitempty"`
	Duration   float64 `json:"duration,omitempty"`
}

func handleSendAudio(player *speech.WebSocketPlayer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req AudioRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Get audio data
		var audioData []byte
		if req.AudioData != "" {
			// Decode base64 audio data
			decoded, err := base64.StdEncoding.DecodeString(req.AudioData)
			if err != nil {
				log.Printf("Failed to decode base64 audio data: %v", err)
				audioData = speech.GetSilentWAV()
			} else {
				audioData = decoded
			}
		} else {
			// Generate silent WAV if no audio data provided
			audioData = speech.GetSilentWAV()
		}

		// Create AudioMeta
		meta := &speech.AudioMeta{
			OriginalText:   req.Text,
			NormalizedText: req.Text,
		}

		// Set duration if provided
		if req.Duration > 0 {
			meta.Duration = time.Duration(req.Duration * float64(time.Second))
		} else {
			// Try to parse WAV duration
			if duration, err := speech.ParseWAVDuration(audioData); err == nil {
				meta.Duration = duration
			} else {
				meta.Duration = 50 * time.Millisecond
			}
		}

		// Play through WebSocket (this sends the message)
		if err := player.Play(audioData, meta); err != nil {
			http.Error(w, "Failed to send audio", http.StatusInternalServerError)
			return
		}

		// Get the ID from the last message (not ideal, but works for testing)
		msgID := uuid.New().String()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"id":      msgID,
		})
	}
}

func handleSendTest(player *speech.WebSocketPlayer) http.HandlerFunc {
	messages := []string{
		"テストメッセージ1: ファイルを読み込んでいます",
		"テストメッセージ2: ビルドを実行中です",
		"テストメッセージ3: テストが完了しました",
		"テストメッセージ4: デプロイを開始します",
		"テストメッセージ5: すべての処理が完了しました",
	}

	index := 0

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Send a test message with audio
		text := messages[index%len(messages)]
		index++

		// Try to load sample.wav from project root
		var audioData []byte
		// Try multiple locations for sample.wav
		samplePaths := []string{
			"sample.wav",
			"../../sample.wav",
			"cmd/test_ws_server/sample.wav",
		}

		var sampleFound bool
		for _, path := range samplePaths {
			if wavData, err := os.ReadFile(path); err == nil {
				audioData = wavData
				log.Printf("Using sample.wav from %s for audio data", path)
				sampleFound = true
				break
			}
		}

		if !sampleFound {
			audioData = speech.GetSilentWAV()
			log.Println("Using silent WAV for audio data (sample.wav not found)")
		}

		// Create AudioMeta
		meta := &speech.AudioMeta{
			OriginalText:   text,
			NormalizedText: text,
		}

		// Parse duration
		if duration, err := speech.ParseWAVDuration(audioData); err == nil {
			meta.Duration = duration
		} else {
			meta.Duration = 50 * time.Millisecond
		}

		// Play through WebSocket
		if err := player.Play(audioData, meta); err != nil {
			http.Error(w, "Failed to send test audio", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"id":      uuid.New().String(),
			"text":    text,
		})
	}
}
