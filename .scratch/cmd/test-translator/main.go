package main

import (
	"fmt"
	"github.com/kazegusuri/claude-companion/narrator"
)

func main() {
	translator := narrator.NewSimpleTranslator()
	normalizer := narrator.NewTextNormalizer()

	testCases := []string{
		"Reading file",
		"Writing file main.go",
		"File not found",
		"Running command: go test",
		"Building project",
		"Found 5 files",
		"3 errors",
		"Completed successfully",
		"ドキュメント「README.md」を読み込みます", // Already Japanese
		"Looking for TODO comments",
	}

	fmt.Println("Translation & Normalization Test:")
	fmt.Println("================================")

	for _, text := range testCases {
		translated := translator.Translate(text)
		normalized := normalizer.Normalize(translated)

		fmt.Printf("Original:    %s\n", text)
		fmt.Printf("Translated:  %s\n", translated)
		fmt.Printf("Normalized:  %s\n", normalized)
		fmt.Println()
	}
}
