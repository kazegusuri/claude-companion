package narrator

import (
	"testing"
)

func TestTextNormalizer_Normalize(t *testing.T) {
	normalizer := NewTextNormalizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic file extension tests
		{
			name:     "Dot Go extension",
			input:    "main.go",
			expected: "mainドットゴー",
		},
		{
			name:     "Dot JS extension",
			input:    "script.js",
			expected: "scriptドットジェーエス",
		},
		{
			name:     "README.md full replacement",
			input:    "README.md",
			expected: "リードミー",
		},
		{
			name:     "README without extension",
			input:    "README",
			expected: "リードミー",
		},

		// Abbreviation tests
		{
			name:     "TODO abbreviation",
			input:    "TODO: fix this",
			expected: "トゥードゥー: fix this",
		},
		{
			name:     "API abbreviation",
			input:    "API endpoint",
			expected: "エーピーアイ endpoint",
		},
		{
			name:     "Multiple abbreviations",
			input:    "TODO: Update API and URL",
			expected: "トゥードゥー: Update エーピーアイ and ユーアールエル",
		},

		// Mixed Japanese and English
		{
			name:     "Japanese with English filename",
			input:    "ファイル main.go を開きます",
			expected: "ファイル mainドットゴー を開きます",
		},
		{
			name:     "Japanese with README.md",
			input:    "README.mdを読み込みます",
			expected: "リードミーを読み込みます",
		},
		{
			name:     "Japanese with abbreviation",
			input:    "APIを使用してTODOリストを取得",
			expected: "エーピーアイを使用してトゥードゥーリストを取得",
		},

		// Dot handling tests
		{
			name:     "Sentence ending dot",
			input:    "This is a sentence.",
			expected: "This is a sentence.",
		},
		{
			name:     "Dot in filename",
			input:    "config.json",
			expected: "configドットジェイソン",
		},
		{
			name:     "Multiple dots",
			input:    "test.min.js",
			expected: "テストドットminドットジェーエス",
		},
		{
			name:     "Dot followed by space",
			input:    "End. Start",
			expected: "End. Start",
		},

		// Hyphen handling tests
		{
			name:     "Hyphenated English word",
			input:    "auto-save feature",
			expected: "auto save feature",
		},
		{
			name:     "Multiple hyphens",
			input:    "user-generated-content",
			expected: "user generated content",
		},

		// Non-ASCII preservation tests
		{
			name:     "Japanese only text",
			input:    "これは日本語のテキストです",
			expected: "これは日本語のテキストです",
		},
		{
			name:     "Emoji preservation",
			input:    "🚀 TODO: Launch app",
			expected: "🚀 トゥードゥー: Launch app",
		},
		{
			name:     "Chinese characters",
			input:    "文件 README.md 已更新",
			expected: "文件 リードミー 已更新",
		},

		// Complex cases
		{
			name:     "Multiple file types",
			input:    "Files: main.go, script.js, style.css",
			expected: "Files: mainドットゴー, scriptドットジェーエス, styleドットシーエスエス",
		},
		{
			name:     "Path with extensions",
			input:    "src/main.go and test/app.js",
			expected: "ソーススラmainドットゴー and テストスラappドットジェーエス",
		},
		{
			name:     "Mixed case abbreviations",
			input:    "api and API are the same",
			expected: "エーピーアイ and エーピーアイ are the same",
		},

		// Domain patterns
		{
			name:     "Simple domain",
			input:    "github.com",
			expected: "ギットハブドットcom",
		},
		{
			name:     "Domain with subdomain",
			input:    "api.github.com",
			expected: "エーピーアイドットギットハブドットcom",
		},
		{
			name:     "Domain with protocol",
			input:    "https://github.com",
			expected: "エイチティーティーピーエス,ギットハブドットcom",
		},
		{
			name:     "Domain with path",
			input:    "github.com/user/repo",
			expected: "ギットハブドットcomスラuserスラrepo",
		},

		// Full path patterns
		{
			name:     "Unix absolute path",
			input:    "/home/user/documents/README.md",
			expected: "スラhomeスラuserスラdocumentsスラリードミー",
		},
		{
			name:     "Windows absolute path",
			input:    "C:\\Users\\Admin\\Documents\\file.txt",
			expected: "C:\\Users\\Admin\\Documents\\fileドットテキスト",
		},
		{
			name:     "Path with multiple directories",
			input:    "/usr/local/bin/npm",
			expected: "スラusrスラlocalスラbinスラエヌピーエム",
		},

		// Relative path patterns
		{
			name:     "Simple relative path",
			input:    "./src/main.go",
			expected: "ドットスラソーススラmainドットゴー",
		},
		{
			name:     "Parent directory path",
			input:    "../lib/utils.js",
			expected: "ドットドットスラライブラリスラutilsドットジェーエス",
		},
		{
			name:     "Nested relative path",
			input:    "../../pkg/models/user.go",
			expected: "ドットドットスラドットドットスラパッケージスラmodelsスラuserドットゴー",
		},

		// Snake case patterns
		{
			name:     "Simple snake case",
			input:    "user_profile.py",
			expected: "user profileドットパイ",
		},
		{
			name:     "Long snake case filename",
			input:    "get_user_profile_by_email_address.js",
			expected: "get user profile by email addressドットジェーエス",
		},
		{
			name:     "Snake case with numbers",
			input:    "api_v2_handler.go",
			expected: "api v2 handlerドットゴー",
		},

		// Kebab case patterns
		{
			name:     "Simple kebab case",
			input:    "user-profile.css",
			expected: "user profileドットシーエスエス",
		},
		{
			name:     "Long kebab case filename",
			input:    "get-user-profile-by-email-address.html",
			expected: "get user profile by email addressドットエイチティーエムエル",
		},
		{
			name:     "Kebab case with numbers",
			input:    "api-v2-handler.yaml",
			expected: "エーピーアイ-v2-handlerドットヤムル",
		},

		// Camel case patterns
		{
			name:     "Simple camel case",
			input:    "UserProfile.java",
			expected: "UserProfileドットjava",
		},
		{
			name:     "Lower camel case",
			input:    "getUserProfile.ts",
			expected: "getUserProfileドットティーエス",
		},
		{
			name:     "Camel case with acronym",
			input:    "APIResponseHandler.go",
			expected: "APIResponseHandlerドットゴー",
		},

		// Mixed patterns
		{
			name:     "Path with snake case file",
			input:    "/home/user/src/database_connection_pool.py",
			expected: "スラhomeスラuserスラソーススラdatabase connection poolドットパイ",
		},
		{
			name:     "URL with kebab case path",
			input:    "https://api.github.com/user-profile/settings",
			expected: "エイチティーティーピーエス,エーピーアイドットギットハブドットcomスラuser profileスラsettings",
		},
		{
			name:     "Complex path with mixed cases",
			input:    "./src/components/UserProfile/user-settings_config.json",
			expected: "ドットスラソーススラcomponentsスラUserProfileスラuser settings configドットジェイソン",
		},

		// Timestamp patterns
		{
			name:     "Timestamp log filename",
			input:    "20251224030405.log",
			expected: "2025 1224 0304 05ドットログ",
		},
		{
			name:     "Timestamp with path",
			input:    "/var/log/app/20251224030405.log",
			expected: "スラvarスラlogスラappスラ2025 1224 0304 05ドットログ",
		},
		{
			name:     "Short number (less than 4 digits)",
			input:    "123.txt",
			expected: "123ドットテキスト",
		},
		{
			name:     "Exactly 4 digits",
			input:    "2025.txt",
			expected: "2025ドットテキスト",
		},
		{
			name:     "5 digits",
			input:    "12345.txt",
			expected: "1234 5ドットテキスト",
		},
		{
			name:     "8 digits",
			input:    "20251224.txt",
			expected: "2025 1224ドットテキスト",
		},

		// Edge cases
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only non-ASCII",
			input:    "日本語のみ",
			expected: "日本語のみ",
		},
		{
			name:     "Only ASCII spaces",
			input:    "   ",
			expected: "   ",
		},
		{
			name:     "Tab and newline preservation",
			input:    "Line1\nLine2\tTODO",
			expected: "Line1\nLine2\tトゥードゥー",
		},

		// Directory name tests
		{
			name:     "src directory",
			input:    "src/components",
			expected: "ソーススラcomponents",
		},
		{
			name:     "node_modules directory",
			input:    "node_modules/package",
			expected: "ノードモジュールスラpackage",
		},

		// Common programming terms
		{
			name:     "npm command",
			input:    "npm install",
			expected: "エヌピーエム install",
		},
		{
			name:     "GitHub reference",
			input:    "Check GitHub for updates",
			expected: "Check ギットハブ for updates",
		},

		// Multiple replacements in one string
		{
			name:     "Complex mixed content",
			input:    "TODO: Update README.md in src/docs folder using GitHub API",
			expected: "トゥードゥー: Update リードミー in ソーススラドキュメント folder using ギットハブ エーピーアイ",
		},
		
		// gRPC tests
		{
			name:     "gRPC lowercase",
			input:    "grpc server",
			expected: "ジーアールピーシー server",
		},
		{
			name:     "gRPC mixed case",
			input:    "gRPC client",
			expected: "ジーアールピーシー client",
		},
		{
			name:     "GRPC uppercase",
			input:    "GRPC protocol",
			expected: "ジーアールピーシー protocol",
		},
		{
			name:     "gRPC in path",
			input:    "/api/grpc/service",
			expected: "スラエーピーアイスラジーアールピーシースラservice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.Normalize(tt.input)
			if result != tt.expected {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
