package speech

import "context"

// Synthesizer interface defines the contract for text-to-speech synthesis
type Synthesizer interface {
	// Synthesize converts text to audio data (WAV format)
	Synthesize(ctx context.Context, text string) ([]byte, error)

	// IsAvailable checks if the synthesizer service is available
	IsAvailable() bool

	// SetVoiceParameters sets voice parameters for synthesis
	SetVoiceParameters(speed, pitch, volume, intonation float64)
}

// Player interface defines the contract for playing audio data
type Player interface {
	// Play plays audio data (WAV format) with metadata
	Play(audioData []byte, meta *AudioMeta) error

	// TestPlay tests if the player is working by playing a silent WAV
	TestPlay() error
}
