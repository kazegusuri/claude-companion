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
			"gRPC":  "ジーアールピーシー",
			"GRPC":  "ジーアールピーシー",

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
	// Extract ASCII printable sequences and apply replacements only to them
	result := ""
	runes := []rune(text)
	i := 0

	for i < len(runes) {
		// Check if current rune is ASCII printable (32-126)
		if runes[i] >= 32 && runes[i] <= 126 {
			// Start of ASCII printable sequence
			start := i
			for i < len(runes) && runes[i] >= 32 && runes[i] <= 126 {
				i++
			}
			// Extract ASCII printable substring
			asciiPart := string(runes[start:i])

			// Apply all normalizations to ASCII part
			normalized := asciiPart

			// First, handle specific full matches like "README.md"
			for old, new := range n.replacements {
				if strings.Contains(old, ".") && len(old) > 3 {
					// Full filename replacements
					normalized = strings.ReplaceAll(normalized, old, new)
				}
			}

			// Handle abbreviations and terms before dots
			normalized = n.replaceAbbreviations(normalized)

			// Replace dots (after abbreviation handling)
			normalized = n.replaceDots(normalized)

			// Handle file extensions after dots have been replaced
			// This will replace patterns like "ドットgo" with "ドットゴー"
			for old, new := range n.replacements {
				if strings.HasPrefix(old, ".") {
					// Convert ".go" to "ドットgo" pattern for matching
					dotPattern := "ドット" + old[1:]
					normalized = strings.ReplaceAll(normalized, dotPattern, new)
				}
			}

			// Replace hyphens in hyphenated English words
			normalized = n.replaceHyphens(normalized)

			// Replace :// with , (before slash replacement)
			normalized = strings.ReplaceAll(normalized, "://", ",")

			// Replace slashes with "スラ"
			normalized = n.replaceSlashes(normalized)

			// Replace underscores with spaces
			normalized = strings.ReplaceAll(normalized, "_", " ")

			// Split long numbers (4+ digits) into groups of 4
			normalized = n.splitLongNumbers(normalized)

			result += normalized
		} else {
			// Non-ASCII printable character, keep as-is
			result += string(runes[i])
			i++
		}
	}

	return result
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

// replaceSlashes replaces forward slashes with "スラ"
func (n *TextNormalizer) replaceSlashes(text string) string {
	return strings.ReplaceAll(text, "/", "スラ")
}

// splitLongNumbers splits numbers with 4 or more digits into groups of 4
func (n *TextNormalizer) splitLongNumbers(text string) string {
	// Pattern to match 4 or more consecutive digits
	re := regexp.MustCompile(`\d{4,}`)

	return re.ReplaceAllStringFunc(text, func(match string) string {
		// Split the number into groups of 4 from the left
		var result []string
		for i := 0; i < len(match); i += 4 {
			end := i + 4
			if end > len(match) {
				end = len(match)
			}
			result = append(result, match[i:end])
		}
		return strings.Join(result, " ")
	})
}
