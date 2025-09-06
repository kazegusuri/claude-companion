package speech

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// VoiceVox handles text-to-speech using VOICEVOX engine
type VoiceVox struct {
	baseURL    string
	speakerID  int
	httpClient *http.Client
	speed      float64
	pitch      float64
	volume     float64
	intonation float64
}

// NewVoiceVox creates a new VOICEVOX synthesizer
func NewVoiceVox(baseURL string, speakerID int) *VoiceVox {
	if baseURL == "" {
		baseURL = "http://localhost:50021"
	}

	return &VoiceVox{
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
func (v *VoiceVox) SetVoiceParameters(speed, pitch, volume, intonation float64) {
	v.speed = speed
	v.pitch = pitch
	v.volume = volume
	v.intonation = intonation
}

// Synthesize converts text to audio data (WAV format)
func (v *VoiceVox) Synthesize(ctx context.Context, text string) ([]byte, error) {
	// Generate audio query
	query, err := v.generateAudioQuery(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to generate audio query: %w", err)
	}

	// Generate audio
	audioData, err := v.generateAudio(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate audio: %w", err)
	}

	return audioData, nil
}

// generateAudioQuery generates audio query from text
func (v *VoiceVox) generateAudioQuery(ctx context.Context, text string) ([]byte, error) {
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
func (v *VoiceVox) generateAudio(ctx context.Context, query []byte) ([]byte, error) {
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

// IsAvailable checks if VOICEVOX server is available
func (v *VoiceVox) IsAvailable() bool {
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
func (v *VoiceVox) GetSpeakers(ctx context.Context) ([]Speaker, error) {
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
