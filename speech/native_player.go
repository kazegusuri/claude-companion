package speech

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// NativePlayer implements Player interface for different platforms
type NativePlayer struct{}

// NewNativePlayer creates a new native audio player
func NewNativePlayer() *NativePlayer {
	return &NativePlayer{}
}

// Play plays audio data using system-specific commands
func (p *NativePlayer) Play(audioData []byte, meta *AudioMeta) error {
	// Log metadata if available
	if meta != nil {
		// Metadata is available for logging or future use
		// For now, we just play the audio
	}

	switch runtime.GOOS {
	case "darwin":
		return p.playMacOS(audioData)
	case "linux":
		return p.playLinux(audioData)
	case "windows":
		return p.playWindows(audioData)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// TestPlay tests if the player is working by playing a silent WAV
func (p *NativePlayer) TestPlay() error {
	// Create minimal metadata for test
	meta := &AudioMeta{
		OriginalText:   "test",
		NormalizedText: "test",
	}

	// Parse duration from silent WAV
	if duration, err := ParseWAVDuration(silentWAV); err == nil {
		meta.Duration = duration
	}

	return p.Play(silentWAV, meta)
}

// playMacOS plays audio on macOS
func (p *NativePlayer) playMacOS(audioData []byte) error {
	// Try ffplay first (supports stdin)
	if _, err := exec.LookPath("ffplay"); err == nil {
		cmd := exec.Command("ffplay", "-nodisp", "-autoexit", "-loglevel", "quiet", "-")
		cmd.Stdin = bytes.NewReader(audioData)
		return cmd.Run()
	}

	// Fall back to afplay with temp file
	tmpFile, err := os.CreateTemp("", "audio_*.wav")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(audioData); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	cmd := exec.Command("afplay", tmpFile.Name())
	return cmd.Run()
}

// playLinux plays audio on Linux
func (p *NativePlayer) playLinux(audioData []byte) error {
	// Try aplay first
	if _, err := exec.LookPath("aplay"); err == nil {
		cmd := exec.Command("aplay", "-q", "-")
		cmd.Stdin = bytes.NewReader(audioData)
		return cmd.Run()
	}

	// Try paplay
	if _, err := exec.LookPath("paplay"); err == nil {
		cmd := exec.Command("paplay")
		cmd.Stdin = bytes.NewReader(audioData)
		return cmd.Run()
	}

	return fmt.Errorf("no audio player found (tried aplay, paplay)")
}

// playWindows plays audio on Windows
func (p *NativePlayer) playWindows(audioData []byte) error {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "audio_*.wav")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

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
