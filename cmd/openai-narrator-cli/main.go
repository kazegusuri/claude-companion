package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kazegusuri/claude-companion/internal/narrator"
)

func main() {
	var (
		apiKey      = flag.String("api-key", "", "OpenAI API key (or set OPENAI_API_KEY env var)")
		text        = flag.String("text", "", "Text to narrate")
		isThinking  = flag.Bool("thinking", false, "Whether this is thinking mode")
		interactive = flag.Bool("i", false, "Interactive mode")
	)
	flag.Parse()

	// Get API key from flag or environment
	key := *apiKey
	if key == "" {
		key = os.Getenv("OPENAI_API_KEY")
	}
	if key == "" {
		fmt.Fprintln(os.Stderr, "Error: OpenAI API key is required. Set OPENAI_API_KEY env var or use -api-key flag")
		os.Exit(1)
	}

	// Create OpenAI narrator
	openAINarrator := narrator.NewOpenAINarrator(key)

	if *interactive {
		// Interactive mode
		fmt.Println("OpenAI Narrator CLI (Interactive Mode)")
		fmt.Println("Type 'quit' or 'exit' to exit")
		fmt.Println("Type 'thinking on' or 'thinking off' to toggle thinking mode")
		fmt.Println()

		scanner := bufio.NewScanner(os.Stdin)
		thinking := false

		for {
			if thinking {
				fmt.Print("thinking> ")
			} else {
				fmt.Print("> ")
			}

			if !scanner.Scan() {
				break
			}

			input := strings.TrimSpace(scanner.Text())
			if input == "" {
				continue
			}

			// Check for special commands
			switch strings.ToLower(input) {
			case "quit", "exit":
				fmt.Println("Goodbye!")
				return
			case "thinking on":
				thinking = true
				fmt.Println("Thinking mode: ON")
				continue
			case "thinking off":
				thinking = false
				fmt.Println("Thinking mode: OFF")
				continue
			}

			// Narrate the text
			result, fallback := openAINarrator.NarrateText(input, thinking, nil)
			if fallback {
				fmt.Println("Fallback:", result)
			} else {
				fmt.Println("Result:", result)
			}
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input:", err)
			os.Exit(1)
		}
	} else {
		// Single text mode
		if *text == "" {
			fmt.Fprintln(os.Stderr, "Error: -text flag is required in non-interactive mode")
			flag.Usage()
			os.Exit(1)
		}

		// Handle stdin input when text is "-"
		inputText := *text
		if inputText == "-" {
			// Read from stdin
			scanner := bufio.NewScanner(os.Stdin)
			var lines []string
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}
			if err := scanner.Err(); err != nil {
				fmt.Fprintln(os.Stderr, "Error reading from stdin:", err)
				os.Exit(1)
			}
			inputText = strings.Join(lines, "\n")
		}

		result, fallback := openAINarrator.NarrateText(inputText, *isThinking, nil)
		if fallback {
			fmt.Fprintln(os.Stderr, "Warning: Fallback mode")
		}
		fmt.Println(result)
	}
}
