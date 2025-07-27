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

	// Handle file extensions in quoted filenames
	normalized = n.normalizeQuotedFilenames(normalized)

	// Then handle abbreviations and terms
	normalized = n.replaceAbbreviations(normalized)

	return normalized
}

// normalizeQuotedFilenames handles filenames in quotes
func (n *TextNormalizer) normalizeQuotedFilenames(text string) string {
	// Match filenames in Japanese quotes 「」
	re := regexp.MustCompile(`「([^」]+\.[a-zA-Z]+)」`)
	text = re.ReplaceAllStringFunc(text, func(match string) string {
		// Extract filename without quotes
		filename := match[3 : len(match)-3] // Remove 「 and 」

		// Check if it's a known full filename
		if replacement, ok := n.replacements[filename]; ok {
			return "「" + replacement + "」"
		}

		// Otherwise, try to replace just the extension
		lastDot := strings.LastIndex(filename, ".")
		if lastDot > 0 {
			name := filename[:lastDot]
			ext := filename[lastDot:]
			if replacement, ok := n.replacements[ext]; ok {
				return "「" + name + replacement + "」"
			}
		}

		return match
	})

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
