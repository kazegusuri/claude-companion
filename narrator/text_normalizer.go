package narrator

import (
	"regexp"
	"strings"
)

// TextNormalizer normalizes text for better TTS pronunciation
type TextNormalizer struct {
	replacements map[string]string
}

// NewTextNormalizer creates a new text normalizer
func NewTextNormalizer() *TextNormalizer {
	return &TextNormalizer{
		replacements: map[string]string{
			// Common file extensions
			"README.md": "リードミー",
			"README":    "リードミー",
			".md":       "ドットエムディー",
			".go":       "ドットゴー",
			".js":       "ドットジェーエス",
			".ts":       "ドットティーエス",
			".py":       "ドットパイ",
			".json":     "ドットジェイソン",
			".yaml":     "ドットヤムル",
			".yml":      "ドットヤムル",
			".txt":      "ドットテキスト",
			".log":      "ドットログ",
			".sh":       "ドットエスエイチ",
			".bash":     "ドットバッシュ",
			".sql":      "ドットエスキューエル",
			".html":     "ドットエイチティーエムエル",
			".css":      "ドットシーエスエス",
			".xml":      "ドットエックスエムエル",

			// Common abbreviations
			"TODO":  "トゥードゥー",
			"API":   "エーピーアイ",
			"URL":   "ユーアールエル",
			"HTTP":  "エイチティーティーピー",
			"HTTPS": "エイチティーティーピーエス",
			"JSON":  "ジェイソン",
			"XML":   "エックスエムエル",
			"CSV":   "シーエスブイ",
			"PDF":   "ピーディーエフ",
			"PNG":   "ピング",
			"JPG":   "ジェイペグ",
			"JPEG":  "ジェイペグ",
			"GIF":   "ジフ",

			// Programming terms
			"npm":        "エヌピーエム",
			"git":        "ギット",
			"GitHub":     "ギットハブ",
			"Docker":     "ドッカー",
			"Kubernetes": "クバネティス",
			"k8s":        "クバネティス",

			// Common directory names
			"src":          "ソース",
			"pkg":          "パッケージ",
			"cmd":          "コマンド",
			"dist":         "ディスト",
			"build":        "ビルド",
			"test":         "テスト",
			"tests":        "テスト",
			"doc":          "ドキュメント",
			"docs":         "ドキュメント",
			"lib":          "ライブラリ",
			"libs":         "ライブラリ",
			"vendor":       "ベンダー",
			"node_modules": "ノードモジュール",
		},
	}
}

// Normalize converts text for better TTS pronunciation
func (n *TextNormalizer) Normalize(text string) string {
	normalized := text

	// First, handle specific full matches like "README.md"
	for old, new := range n.replacements {
		if strings.Contains(old, ".") && len(old) > 3 {
			// Full filename replacements
			normalized = strings.ReplaceAll(normalized, old, new)
		}
	}

	// Replace dots first (before file extension handling)
	normalized = n.replaceDots(normalized)

	// Handle file extensions in quoted filenames
	normalized = n.normalizeQuotedFilenames(normalized)

	// Then handle abbreviations and terms
	normalized = n.replaceAbbreviations(normalized)

	// Replace hyphens in hyphenated English words
	normalized = n.replaceHyphens(normalized)

	return normalized
}

// normalizeQuotedFilenames handles filenames in quotes
func (n *TextNormalizer) normalizeQuotedFilenames(text string) string {
	// Since dots are already replaced, we don't need to handle them here
	return text
}

// replaceAbbreviations replaces common abbreviations and terms
func (n *TextNormalizer) replaceAbbreviations(text string) string {
	// Create a regex pattern for word boundaries
	for old, new := range n.replacements {
		if !strings.Contains(old, ".") {
			// Use word boundary regex for abbreviations
			pattern := `\b` + regexp.QuoteMeta(old) + `\b`
			re := regexp.MustCompile("(?i)" + pattern)
			text = re.ReplaceAllString(text, new)
		}
	}
	return text
}

// replaceDots replaces dots that are not sentence endings
func (n *TextNormalizer) replaceDots(text string) string {
	result := ""
	runes := []rune(text)

	for i := 0; i < len(runes); i++ {
		if runes[i] == '.' {
			// Check if this is a sentence ending
			isSentenceEnd := false

			// Check what follows the dot
			if i == len(runes)-1 {
				// Dot at end of text
				isSentenceEnd = true
			} else if i+1 < len(runes) && (runes[i+1] == ' ' || runes[i+1] == '　' || runes[i+1] == '\n' || runes[i+1] == '\t') {
				// Dot followed by whitespace (including full-width space)
				isSentenceEnd = true
			}

			if isSentenceEnd {
				result += "."
			} else {
				result += "ドット"
			}
		} else {
			result += string(runes[i])
		}
	}

	return result
}

// replaceHyphens replaces hyphens in hyphenated English words with spaces
func (n *TextNormalizer) replaceHyphens(text string) string {
	// Pattern: hyphen between two alphabetic characters (English words)
	re := regexp.MustCompile(`([a-zA-Z])-([a-zA-Z])`)
	return re.ReplaceAllString(text, "$1 $2")
}
