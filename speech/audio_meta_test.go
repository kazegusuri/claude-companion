package speech

import (
	"testing"
	"time"
)

func TestParseWAVDuration(t *testing.T) {
	tests := []struct {
		name      string
		audioData []byte
		wantErr   bool
	}{
		{
			name:      "Valid silent WAV",
			audioData: silentWAV,
			wantErr:   false,
		},
		{
			name:      "Too short data",
			audioData: []byte{1, 2, 3},
			wantErr:   true,
		},
		{
			name:      "Invalid RIFF header",
			audioData: make([]byte, 44),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDuration, err := ParseWAVDuration(tt.audioData)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseWAVDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Just check that we got a valid duration
				if gotDuration <= 0 {
					t.Errorf("ParseWAVDuration() returned invalid duration: %v", gotDuration)
				}
				// Log the actual duration for debugging
				t.Logf("ParseWAVDuration() returned: %v", gotDuration)
			}
		})
	}
}

func TestAudioMeta(t *testing.T) {
	meta := &AudioMeta{
		OriginalText:   "Hello, World!",
		NormalizedText: "hello world",
		Duration:       5 * time.Second,
	}

	if meta.OriginalText != "Hello, World!" {
		t.Errorf("OriginalText = %v, want %v", meta.OriginalText, "Hello, World!")
	}

	if meta.NormalizedText != "hello world" {
		t.Errorf("NormalizedText = %v, want %v", meta.NormalizedText, "hello world")
	}

	if meta.Duration != 5*time.Second {
		t.Errorf("Duration = %v, want %v", meta.Duration, 5*time.Second)
	}
}
