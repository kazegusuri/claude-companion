package speech

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-audio/wav"
)

// AudioMeta contains metadata about audio data
type AudioMeta struct {
	// OriginalText is the text before normalization
	OriginalText string

	// NormalizedText is the text after normalization
	NormalizedText string

	// Duration is the duration of the audio
	Duration time.Duration
}

// ParseWAVDuration parses the duration from WAV audio data using go-audio/wav library
func ParseWAVDuration(audioData []byte) (time.Duration, error) {
	// Create a reader from the audio data
	reader := bytes.NewReader(audioData)

	// Create a WAV decoder
	decoder := wav.NewDecoder(reader)

	// Read the WAV format information
	if !decoder.IsValidFile() {
		return 0, fmt.Errorf("invalid WAV file")
	}

	// Get format information
	format := decoder.Format()
	if format == nil {
		return 0, fmt.Errorf("could not read WAV format")
	}

	// Get sample rate
	sampleRate := format.SampleRate
	if sampleRate == 0 {
		return 0, fmt.Errorf("invalid sample rate: 0")
	}

	// Try to get duration directly
	// Note: Duration() returns (time.Duration, error)
	duration, err := decoder.Duration()
	if err != nil || duration == 0 {
		// If duration is not available, calculate it manually
		// Get PCM info if available
		if decoder.PCMLen() > 0 {
			// PCMLen returns the total number of frames
			// Duration = frames / sampleRate
			frames := decoder.PCMLen()
			seconds := float64(frames) / float64(sampleRate)
			duration = time.Duration(seconds * float64(time.Second))
		} else {
			// Fallback: calculate from PCMSize
			// PCMSize is a field, not a method
			dataSize := decoder.PCMSize
			if dataSize > 0 && format.NumChannels > 0 {
				// Calculate bytes per frame
				// BitDepth is the correct field name
				bytesPerSample := uint32(decoder.BitDepth / 8)
				bytesPerFrame := bytesPerSample * uint32(format.NumChannels)

				if bytesPerFrame > 0 {
					frames := uint32(dataSize) / bytesPerFrame
					seconds := float64(frames) / float64(sampleRate)
					duration = time.Duration(seconds * float64(time.Second))
				}
			}
		}
	}

	if duration == 0 {
		return 0, fmt.Errorf("could not determine audio duration")
	}

	return duration, nil
}
