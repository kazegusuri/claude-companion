package narrator

import (
	"regexp"
	"strings"
)

// SimpleTranslator provides basic English to Japanese translation for common phrases
type SimpleTranslator struct {
	phrases map[string]string
}

// NewSimpleTranslator creates a new translator
func NewSimpleTranslator() *SimpleTranslator {
	return &SimpleTranslator{
		phrases: map[string]string{
			// Common programming phrases
			"Reading file":    "ファイルを読み込みます",
			"Writing file":    "ファイルを書き込みます",
			"Creating file":   "ファイルを作成します",
			"Editing file":    "ファイルを編集します",
			"Deleting file":   "ファイルを削除します",
			"Running command": "コマンドを実行します",
			"Executing":       "実行します",
			"Building":        "ビルドします",
			"Testing":         "テストします",
			"Installing":      "インストールします",
			"Searching":       "検索します",
			"Looking for":     "探します",
			"Checking":        "確認します",
			"Analyzing":       "分析します",
			"Processing":      "処理します",
			"Completed":       "完了しました",
			"Failed":          "失敗しました",
			"Error":           "エラー",
			"Warning":         "警告",
			"Success":         "成功",
			"Done":            "完了",

			// File operations
			"file not found":      "ファイルが見つかりません",
			"directory not found": "ディレクトリが見つかりません",
			"permission denied":   "権限がありません",
			"already exists":      "既に存在します",
			"cannot read":         "読み込めません",
			"cannot write":        "書き込めません",

			// Common responses
			"Yes":         "はい",
			"No":          "いいえ",
			"OK":          "OK",
			"Cancel":      "キャンセル",
			"Continue":    "続行",
			"Stop":        "停止",
			"Please wait": "お待ちください",
			"Loading":     "読み込み中",
			"Saving":      "保存中",

			// Time-related
			"now":         "今",
			"today":       "今日",
			"yesterday":   "昨日",
			"tomorrow":    "明日",
			"seconds ago": "秒前",
			"minutes ago": "分前",
			"hours ago":   "時間前",
			"days ago":    "日前",
		},
	}
}

// Translate attempts to translate English text to Japanese
func (t *SimpleTranslator) Translate(text string) string {
	// Check if text is already mostly Japanese
	if t.isMostlyJapanese(text) {
		return text
	}

	translated := text

	// First try exact phrase matching (case-insensitive)
	lowerText := strings.ToLower(text)
	for eng, jpn := range t.phrases {
		if strings.ToLower(eng) == lowerText {
			return jpn
		}
	}

	// Then try partial matching for common patterns
	for eng, jpn := range t.phrases {
		// Case-insensitive replacement
		re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(eng))
		translated = re.ReplaceAllString(translated, jpn)
	}

	// Translate common patterns
	translated = t.translatePatterns(translated)

	return translated
}

// isMostlyJapanese checks if text contains Japanese characters
func (t *SimpleTranslator) isMostlyJapanese(text string) bool {
	// Count Japanese characters (Hiragana, Katakana, Kanji)
	japaneseCount := 0
	totalCount := 0

	for _, r := range text {
		if r >= 'A' && r <= 'z' {
			totalCount++
		} else if (r >= 0x3040 && r <= 0x309F) || // Hiragana
			(r >= 0x30A0 && r <= 0x30FF) || // Katakana
			(r >= 0x4E00 && r <= 0x9FAF) { // Kanji
			japaneseCount++
			totalCount++
		}
	}

	if totalCount == 0 {
		return true // No alphabetic characters, assume it's OK
	}

	// If more than 50% is Japanese, consider it mostly Japanese
	return float64(japaneseCount)/float64(totalCount) > 0.5
}

// translatePatterns handles common patterns
func (t *SimpleTranslator) translatePatterns(text string) string {
	// Pattern: "Reading X" -> "Xを読み込みます"
	patterns := []struct {
		pattern     string
		replacement string
	}{
		// File operations with specific patterns
		{`(?i)^reading\s+file\s*$`, "ファイルを読み込みます"},
		{`(?i)^writing\s+file\s+(.+)$`, "$1を書き込みます"},
		{`(?i)^reading\s+(.+)$`, "$1を読み込みます"},
		{`(?i)^writing\s+(.+)$`, "$1を書き込みます"},
		{`(?i)^creating\s+(.+)$`, "$1を作成します"},
		{`(?i)^running\s+command:\s*(.+)$`, "コマンド「$1」を実行します"},
		{`(?i)^running\s+(.+)$`, "$1を実行します"},
		{`(?i)^executing\s+(.+)$`, "$1を実行します"},
		{`(?i)^building\s+(.+)$`, "$1をビルドします"},
		{`(?i)^testing\s+(.+)$`, "$1をテストします"},
		{`(?i)^searching\s+for\s+(.+)$`, "$1を検索します"},
		{`(?i)^looking\s+for\s+(.+)$`, "$1を探します"},

		// Results patterns
		{`(?i)^found\s+(\d+)\s+files?$`, "$1個のファイルを見つけました"},
		{`(?i)^found\s+(\d+)\s+matches?$`, "$1個の一致を見つけました"},
		{`(?i)^(\d+)\s+errors?$`, "$1個のエラー"},
		{`(?i)^(\d+)\s+warnings?$`, "$1個の警告"},

		// Status patterns
		{`(?i)^completed\s+successfully$`, "正常に完了しました"},
		{`(?i)^failed\s+to\s+(.+)$`, "$1に失敗しました"},
		{`(?i)^cannot\s+(.+)$`, "$1できません"},

		// Common suffixes
		{`(?i)\s+successfully$`, ""},
		{`(?i)\s+failed$`, "が失敗しました"},
	}

	result := text
	for _, p := range patterns {
		re := regexp.MustCompile(p.pattern)
		if re.MatchString(result) {
			result = re.ReplaceAllString(result, p.replacement)
			break // Apply only the first matching pattern
		}
	}

	// Clean up any remaining English words
	result = t.cleanupEnglish(result)

	return result
}

// cleanupEnglish removes common English words that weren't translated
func (t *SimpleTranslator) cleanupEnglish(text string) string {
	// Remove common English words at the end
	suffixes := []string{
		"successfully",
		"failed",
		"completed",
		"finished",
		"done",
	}

	for _, suffix := range suffixes {
		re := regexp.MustCompile(`(?i)\s+` + suffix + `$`)
		text = re.ReplaceAllString(text, "")
	}

	return strings.TrimSpace(text)
}
