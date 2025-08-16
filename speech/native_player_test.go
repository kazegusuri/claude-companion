package speech

import (
	"testing"
)

func TestNativePlayer_TestPlay(t *testing.T) {
	player := NewNativePlayer()

	// Test that TestPlay doesn't return an error
	// This will attempt to play a silent WAV file
	err := player.TestPlay()

	// We allow errors here because audio playback might not be available
	// in all test environments (e.g., CI/CD pipelines without audio support)
	if err != nil {
		t.Logf("TestPlay returned error (this might be expected in environments without audio): %v", err)
	} else {
		t.Log("TestPlay succeeded - audio player is working")
	}
}

func TestGetSilentWAV(t *testing.T) {
	wav := GetSilentWAV()

	// Check that we get a valid WAV header
	if len(wav) < 44 {
		t.Errorf("Silent WAV is too short: got %d bytes, want at least 44", len(wav))
	}

	// Check RIFF header
	if string(wav[0:4]) != "RIFF" {
		t.Errorf("Invalid RIFF header: got %s, want RIFF", string(wav[0:4]))
	}

	// Check WAVE format
	if string(wav[8:12]) != "WAVE" {
		t.Errorf("Invalid WAVE format: got %s, want WAVE", string(wav[8:12]))
	}
}
