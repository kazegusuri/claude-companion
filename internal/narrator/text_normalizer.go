package narrator

import (
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

const ellipsisMarker = "なんとか,"

// TextNormalizer normalizes text for better TTS pronunciation
type TextNormalizer struct {
	replacements       map[string]string
	domainReplacements map[string]string
}

// NewTextNormalizer creates a new text normalizer
func NewTextNormalizer() *TextNormalizer {
	return &TextNormalizer{
		domainReplacements: map[string]string{
			"github.com":        "ギットハブ",
			"api.github.com":    "ギットハブAPI",
			"google.com":        "グーグル",
			"youtube.com":       "ユーチューブ",
			"twitter.com":       "ツイッター",
			"x.com":             "エックス",
			"facebook.com":      "フェイスブック",
			"instagram.com":     "インスタグラム",
			"linkedin.com":      "リンクトイン",
			"reddit.com":        "レディット",
			"stackoverflow.com": "スタックオーバーフロー",
			"amazon.com":        "アマゾン",
			"wikipedia.org":     "ウィキペディア",
			"openai.com":        "オープンエーアイ",
			"anthropic.com":     "アンソロピック",
			"microsoft.com":     "マイクロソフト",
			"apple.com":         "アップル",
			"golang.org":        "ゴーラング",
			"nodejs.org":        "ノードジェーエス",
			"python.org":        "パイソン",
			"npmjs.com":         "エヌピーエム",
			"docker.com":        "ドッカー",
			"kubernetes.io":     "クバネティス",
		},
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

			// Check if the ASCII part is a URL or path
			skipNormalProcessing := false
			if parsedURL, err := url.Parse(asciiPart); err == nil && parsedURL.Host != "" {
				// It's a URL with a host
				if replacement, ok := n.domainReplacements[parsedURL.Host]; ok {
					// Replace the entire URL with the domain-specific replacement
					normalized = replacement
					skipNormalProcessing = true
				} else {
					// Domain not in replacements, abbreviate to "host ドメイン"
					normalized = parsedURL.Host + " ドメイン"
					skipNormalProcessing = true
				}
			} else if n.isPath(asciiPart) {
				// It's a path, abbreviate if needed
				normalized = n.abbreviatePath(asciiPart)
				// Continue with normal processing for the abbreviated path
				skipNormalProcessing = false
			} else if !strings.Contains(asciiPart, "/") && n.isLongFilename(asciiPart) {
				// It's a long filename (no path), abbreviate if needed
				normalized = n.abbreviateLongFilename(asciiPart)
				// Continue with normal processing for the abbreviated filename
				skipNormalProcessing = false
			} else if strings.Contains(asciiPart, "/") {
				// It's a path but not long enough to abbreviate the path itself
				// Check if the filename part is long and needs abbreviation
				lastSlash := strings.LastIndex(asciiPart, "/")
				if lastSlash >= 0 && lastSlash < len(asciiPart)-1 {
					filename := asciiPart[lastSlash+1:]
					if n.isLongFilename(filename) {
						// Long filename in a path
						pathPart := asciiPart[:lastSlash+1]
						abbreviatedFilename := n.abbreviateLongFilename(filename)
						normalized = pathPart + abbreviatedFilename
						skipNormalProcessing = false
					}
				}
			}

			// Apply normal replacements if not skipped
			if !skipNormalProcessing {
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
			}

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

// isPath checks if the string looks like a file path that should be abbreviated
func (n *TextNormalizer) isPath(text string) bool {
	// Don't treat as path if it's a URL
	if strings.Contains(text, "://") {
		return false
	}

	// Don't abbreviate paths starting with ..
	if strings.HasPrefix(text, "..") {
		return false
	}

	// Count path segments
	cleanPath := strings.TrimPrefix(text, "./")
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	parts := strings.Split(cleanPath, "/")

	// Filter out empty parts
	var nonEmptyParts []string
	for _, part := range parts {
		if part != "" && part != "." {
			nonEmptyParts = append(nonEmptyParts, part)
		}
	}

	// Only abbreviate if there are more than 4 parts
	// (5 or more parts means we can safely abbreviate to 3 parts)
	return len(nonEmptyParts) > 4
}

// abbreviatePath shortens a path to show only the first and last parts with ellipsis
func (n *TextNormalizer) abbreviatePath(path string) string {
	// Remove ./ prefix if present
	if strings.HasPrefix(path, "./") {
		path = path[2:]
	}

	// Check if it's an absolute path
	isAbsolute := strings.HasPrefix(path, "/")

	// Clean the path
	path = filepath.Clean(path)

	// Split the path into parts
	parts := strings.Split(path, "/")

	// Filter out empty parts
	var nonEmptyParts []string
	for _, part := range parts {
		if part != "" && part != "." {
			nonEmptyParts = append(nonEmptyParts, part)
		}
	}

	// If 3 or fewer parts, return as is (but without ./ prefix)
	if len(nonEmptyParts) <= 3 {
		if isAbsolute {
			return "/" + strings.Join(nonEmptyParts, "/")
		}
		return strings.Join(nonEmptyParts, "/")
	}

	// If 5 or more parts, show first part + ellipsis + last 3 parts
	// For absolute paths: /first/...省略.../parent2/parent/file
	// For relative paths: first/...省略.../parent2/parent/file
	if len(nonEmptyParts) >= 5 {
		// Check if the filename part is long and needs abbreviation
		filename := nonEmptyParts[len(nonEmptyParts)-1]
		if n.isLongFilename(filename) {
			// Abbreviate both path and filename
			abbreviatedFilename := n.abbreviateLongFilename(filename)
			result := []string{nonEmptyParts[0], ellipsisMarker, nonEmptyParts[len(nonEmptyParts)-3], nonEmptyParts[len(nonEmptyParts)-2], abbreviatedFilename}
			if isAbsolute {
				return "/" + strings.Join(result, "/")
			}
			return strings.Join(result, "/")
		} else {
			// Only abbreviate path
			result := []string{nonEmptyParts[0], ellipsisMarker, nonEmptyParts[len(nonEmptyParts)-3], nonEmptyParts[len(nonEmptyParts)-2], nonEmptyParts[len(nonEmptyParts)-1]}
			if isAbsolute {
				return "/" + strings.Join(result, "/")
			}
			return strings.Join(result, "/")
		}
	}

	// For 4 parts, return as is
	if isAbsolute {
		return "/" + strings.Join(nonEmptyParts, "/")
	}
	return strings.Join(nonEmptyParts, "/")
}

// isLongFilename checks if the string is a long filename that should be abbreviated
// Note: This should be called with just the filename part, not a full path
func (n *TextNormalizer) isLongFilename(text string) bool {
	// Must have a file extension to be considered a filename
	if !strings.Contains(text, ".") {
		return false
	}

	// Don't treat as filename if it contains spaces (likely a sentence)
	if strings.Contains(text, " ") {
		return false
	}

	// Split by underscores, hyphens, or CamelCase
	words := n.splitIntoWords(text)

	// Only abbreviate if there are more than 5 words
	return len(words) > 5
}

// splitIntoWords splits a filename into words by snake_case, kebab-case, or CamelCase
func (n *TextNormalizer) splitIntoWords(text string) []string {
	// First, handle file extension
	lastDot := strings.LastIndex(text, ".")
	name := text
	if lastDot > 0 {
		name = text[:lastDot]
	}

	// Replace underscores and hyphens with spaces
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")

	// Split CamelCase
	// Insert spaces before uppercase letters that follow lowercase letters
	re := regexp.MustCompile(`([a-z])([A-Z])`)
	name = re.ReplaceAllString(name, "$1 $2")

	// Split by spaces and filter empty strings
	parts := strings.Fields(name)

	return parts
}

// abbreviateLongFilename shortens a long filename to show first 2 and last 3 words
func (n *TextNormalizer) abbreviateLongFilename(filename string) string {
	// Split filename and extension
	lastDot := strings.LastIndex(filename, ".")
	name := filename
	ext := ""
	if lastDot > 0 {
		name = filename[:lastDot]
		ext = filename[lastDot:]
	}

	// Split into words
	words := n.splitIntoWords(name)

	// If 5 or fewer words, return as is (with underscores/hyphens replaced)
	if len(words) <= 5 {
		return strings.Join(words, "_") + ext
	}

	// Show first 2 words + ellipsis + last 3 words
	result := []string{
		words[0],
		words[1],
		ellipsisMarker,
		words[len(words)-3],
		words[len(words)-2],
		words[len(words)-1],
	}

	return strings.Join(result, "_") + ext
}
