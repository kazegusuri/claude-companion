package narrator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// VoiceVoxClient handles text-to-speech using VOICEVOX engine
type VoiceVoxClient struct {
	baseURL    string
	speakerID  int
	httpClient *http.Client
	speed      float64
	pitch      float64
	volume     float64
	intonation float64
}

// NewVoiceVoxClient creates a new VOICEVOX client
func NewVoiceVoxClient(baseURL string, speakerID int) *VoiceVoxClient {
	if baseURL == "" {
		baseURL = "http://localhost:50021"
	}

	return &VoiceVoxClient{
		baseURL:   baseURL,
		speakerID: speakerID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		speed:      1.5,
		pitch:      0.0,
		volume:     1.0,
		intonation: 1.0,
	}
}

// SetVoiceParameters sets voice parameters
func (v *VoiceVoxClient) SetVoiceParameters(speed, pitch, volume, intonation float64) {
	v.speed = speed
	v.pitch = pitch
	v.volume = volume
	v.intonation = intonation
}

// TextToSpeech converts text to speech and plays it
func (v *VoiceVoxClient) TextToSpeech(ctx context.Context, text string) error {
	// Generate audio query
	query, err := v.generateAudioQuery(ctx, text)
	if err != nil {
		return fmt.Errorf("failed to generate audio query: %w", err)
	}

	// Generate audio
	audioData, err := v.generateAudio(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to generate audio: %w", err)
	}

	// Play audio
	return v.playAudio(audioData)
}

// generateAudioQuery generates audio query from text
func (v *VoiceVoxClient) generateAudioQuery(ctx context.Context, text string) ([]byte, error) {
	params := url.Values{}
	params.Add("text", text)
	params.Add("speaker", fmt.Sprintf("%d", v.speakerID))

	url := fmt.Sprintf("%s/audio_query?%s", v.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("audio query failed: %s, body: %s", resp.Status, string(body))
	}

	queryData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Apply voice parameters
	var query map[string]interface{}
	if err := json.Unmarshal(queryData, &query); err != nil {
		return nil, err
	}

	query["speedScale"] = v.speed
	query["pitchScale"] = v.pitch
	query["volumeScale"] = v.volume
	query["intonationScale"] = v.intonation

	return json.Marshal(query)
}

// generateAudio generates audio from query
func (v *VoiceVoxClient) generateAudio(ctx context.Context, query []byte) ([]byte, error) {
	params := url.Values{}
	params.Add("speaker", fmt.Sprintf("%d", v.speakerID))

	url := fmt.Sprintf("%s/synthesis?%s", v.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(query))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("synthesis failed: %s, body: %s", resp.Status, string(body))
	}

	return io.ReadAll(resp.Body)
}

// playAudio plays audio data using system command
func (v *VoiceVoxClient) playAudio(audioData []byte) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		// macOS: try ffplay first (supports stdin), then fall back to afplay with temp file
		if _, err := exec.LookPath("ffplay"); err == nil {
			cmd = exec.Command("ffplay", "-nodisp", "-autoexit", "-loglevel", "quiet", "-")
		} else {
			// ffplay not found, use temp file approach for afplay
			return v.playAudioMacOS(audioData)
		}
	case "linux":
		// Linux: try aplay first, then paplay
		if _, err := exec.LookPath("aplay"); err == nil {
			cmd = exec.Command("aplay", "-q", "-")
		} else if _, err := exec.LookPath("paplay"); err == nil {
			cmd = exec.Command("paplay")
		} else {
			return fmt.Errorf("no audio player found (tried aplay, paplay)")
		}
	case "windows":
		// Windows: use PowerShell
		// This creates a temporary file approach since piping binary data to PowerShell is complex
		return v.playAudioWindows(audioData)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	cmd.Stdin = bytes.NewReader(audioData)
	return cmd.Run()
}

// playAudioMacOS handles macOS-specific audio playback using temp file
func (v *VoiceVoxClient) playAudioMacOS(audioData []byte) error {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "voicevox_*.wav")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	// Write audio data
	if _, err := tmpFile.Write(audioData); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	// Play using afplay
	cmd := exec.Command("afplay", tmpFile.Name())
	return cmd.Run()
}

// playAudioWindows handles Windows-specific audio playback
func (v *VoiceVoxClient) playAudioWindows(audioData []byte) error {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "voicevox_*.wav")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	// Write audio data
	if _, err := tmpFile.Write(audioData); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	// Play using PowerShell
	cmd := exec.Command("powershell", "-Command",
		fmt.Sprintf("(New-Object Media.SoundPlayer '%s').PlaySync()", tmpFile.Name()))
	return cmd.Run()
}

// IsAvailable checks if VOICEVOX server is available
func (v *VoiceVoxClient) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", v.baseURL+"/version", nil)
	if err != nil {
		return false
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetSpeakers returns available speakers
func (v *VoiceVoxClient) GetSpeakers(ctx context.Context) ([]Speaker, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", v.baseURL+"/speakers", nil)
	if err != nil {
		return nil, err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get speakers: %s", resp.Status)
	}

	var speakers []Speaker
	if err := json.NewDecoder(resp.Body).Decode(&speakers); err != nil {
		return nil, err
	}

	return speakers, nil
}

// Speaker represents a VOICEVOX speaker
type Speaker struct {
	Name   string         `json:"name"`
	Styles []SpeakerStyle `json:"styles"`
}

// SpeakerStyle represents a speaker's style
type SpeakerStyle struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}
