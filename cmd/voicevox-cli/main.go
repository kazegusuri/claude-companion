package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kazegusuri/claude-companion/narrator"
)

func main() {
	var (
		baseURL    = flag.String("url", "http://localhost:50021", "VoiceVox API base URL")
		speakerID  = flag.Int("speaker", 3, "Speaker ID")
		speed      = flag.Float64("speed", 1.5, "Speech speed")
		pitch      = flag.Float64("pitch", 0.0, "Voice pitch")
		volume     = flag.Float64("volume", 1.0, "Voice volume")
		intonation = flag.Float64("intonation", 1.0, "Voice intonation")
	)
	flag.Parse()

	// Create VoiceVox client
	client := narrator.NewVoiceVoxClient(*baseURL, *speakerID)
	client.SetVoiceParameters(*speed, *pitch, *volume, *intonation)

	// Check if VoiceVox is available
	if !client.IsAvailable() {
		log.Fatalf("VoiceVox server is not available at %s", *baseURL)
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
		if err := client.TextToSpeech(ctx, normalizedText); err != nil {
			log.Printf("Error: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading stdin: %v", err)
	}
}
