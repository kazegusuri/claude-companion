package main

import (
	"fmt"
	"github.com/kazegusuri/claude-companion/narrator"
)

func main() {
	normalizer := narrator.NewTextNormalizer()

	testCases := []string{
		"ドキュメント「README.md」を読み込みます",
		"Goファイル「main.go」を読み込みます",
		"APIドキュメントを参照します",
		"HTTPSでGitHubにアクセスします",
		"TODO: npmでパッケージをインストール",
		"srcディレクトリのtest.jsを確認",
	}

	fmt.Println("Text Normalizer Test:")
	fmt.Println("====================")

	for _, text := range testCases {
		normalized := normalizer.Normalize(text)
		fmt.Printf("Original:   %s\n", text)
		fmt.Printf("Normalized: %s\n", normalized)
		fmt.Println()
	}
}
