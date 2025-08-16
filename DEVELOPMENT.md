# Development

## Go Backend Development

### Prerequisites
- Go 1.19 or higher
- Make

### Building
```bash
make build    # Build the binary
make test     # Run tests
make fmt      # Format code
make clean    # Clean build artifacts
```

### Project Structure (Go)
```
.
├── main.go                      # Main application and CLI
├── event/
│   ├── event.go                # Event type definitions
│   ├── parser.go               # Event parsing logic
│   ├── formatter.go            # Event formatting
│   ├── handler.go              # Event handling and routing
│   ├── session_watcher.go      # Individual session file watcher
│   ├── projects_watcher.go     # Projects directory watcher
│   ├── notification_watcher.go # Notification log watcher
│   └── session_file_manager.go # Session watcher lifecycle management
├── narrator/
│   ├── narrator.go             # Narrator interface and base implementation
│   ├── config_narrator.go      # Configuration-based narrator
│   ├── voice_narrator.go       # Voice output implementation
│   ├── priority_queue.go       # Priority queue for voice messages
│   └── voicevox.go            # VOICEVOX client
├── Makefile                    # Build automation
├── CLAUDE.md                   # Instructions for Claude
└── README.md                   # Main documentation
```

## Web Frontend Development

### Prerequisites
- [Bun](https://bun.sh/) v1.0 以上
- Node.js 18+ (互換性のため)

### Development Commands

```bash
# 依存関係のインストール
bun install

# 開発サーバーの起動
bun run dev

# ビルド
bun run build

# リントチェック
bun run lint

# フォーマット
bun run format

# リント & フォーマットチェック
bun run check

# テスト
bun test
```

### Project Structure (Web)

```
src/
├── components/
│   └── Live2DViewer.tsx  # Live2D ビューアコンポーネント
├── App.tsx               # メインアプリケーション
└── main.tsx             # エントリーポイント

public/
└── live2d/
    ├── core/            # Live2D Cubism Core
    └── models/          # Live2D モデルファイル
```

### 技術スタック

#### Backend (Go)
- Go 1.19+
- fsnotify (ファイル監視)
- VOICEVOX (音声合成)
- OpenAI API (AIナレーター)

#### Frontend (Web)
- **ランタイム**: Bun
- **フレームワーク**: React 18
- **ビルドツール**: Vite
- **言語**: TypeScript (厳密モード)
- **リンター/フォーマッター**: Biome
- **Live2D**: pixi-live2d-display

## Contributing

1. Follow the coding standards in CLAUDE.md
2. For Go code: Run `make fmt` before committing
3. For Web code: Run `bun run check` before committing
4. Ensure all tests pass:
   - Go: `make test`
   - Web: `bun test`
5. Write clear commit messages