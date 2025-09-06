package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kazegusuri/claude-companion/internal/narrator"
	"github.com/kazegusuri/claude-companion/internal/speech"
)

func main() {
	var (
		baseURL      = flag.String("url", "http://localhost:50021", "VoiceVox API base URL")
		speakerID    = flag.Int("speaker", 3, "Speaker ID")
		speed        = flag.Float64("speed", 1.5, "Speech speed")
		pitch        = flag.Float64("pitch", 0.0, "Voice pitch")
		volume       = flag.Float64("volume", 1.0, "Voice volume")
		intonation   = flag.Float64("intonation", 1.0, "Voice intonation")
		listSpeakers = flag.Bool("list-speakers", false, "List available speakers and exit")
	)
	flag.Parse()

	// Create VoiceVox synthesizer and audio player
	synthesizer := speech.NewVoiceVox(*baseURL, *speakerID)
	synthesizer.SetVoiceParameters(*speed, *pitch, *volume, *intonation)
	player := speech.NewNativePlayer()

	// Check if VoiceVox is available
	if !synthesizer.IsAvailable() {
		log.Fatalf("VoiceVox server is not available at %s", *baseURL)
	}

	// List speakers if requested
	if *listSpeakers {
		ctx := context.Background()
		speakers, err := synthesizer.GetSpeakers(ctx)
		if err != nil {
			log.Fatalf("Failed to get speakers: %v", err)
		}

		fmt.Println("Available VoiceVox speakers:")
		fmt.Println("ID | Name | Style")
		fmt.Println("---|------|------")
		for _, speaker := range speakers {
			for _, style := range speaker.Styles {
				fmt.Printf("%2d | %s | %s\n", style.ID, speaker.Name, style.Name)
			}
		}
		return
	}

	// Create text normalizer
	normalizer := narrator.NewTextNormalizer()

	// Read from stdin
	scanner := bufio.NewScanner(os.Stdin)
	ctx := context.Background()

	for scanner.Scan() {
		text := scanner.Text()
		if text == "" {
			continue
		}

		// Normalize text for better pronunciation
		normalizedText := normalizer.Normalize(text)

		fmt.Printf("Original: %s\n", text)
		fmt.Printf("Speaking: %s\n", normalizedText)

		// Synthesize and play
		audioData, err := synthesizer.Synthesize(ctx, normalizedText)
		if err != nil {
			log.Printf("Error synthesizing: %v", err)
			continue
		}

		// Create metadata
		meta := &speech.AudioMeta{
			OriginalText:   text,
			NormalizedText: normalizedText,
		}

		// Parse audio duration
		if duration, err := speech.ParseWAVDuration(audioData); err == nil {
			meta.Duration = duration
			fmt.Printf("Duration: %v\n", duration)
		} else {
			log.Printf("Warning: Failed to parse WAV duration: %v", err)
		}

		if err := player.Play(audioData, meta); err != nil {
			log.Printf("Error playing: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading stdin: %v", err)
	}
}
