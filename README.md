# Claude Companion

ClaudeのJSONLログファイルをリアルタイムで解析・表示するツールです。構造化されたログイベントを解析し、人間が読みやすい形式で表示し、音声ナレーション機能も提供します。

**注意**: これは趣味のプロジェクトです。インターフェースや機能は予告なく変更される可能性があります。

[English README](README.en.md)

## 機能

- **リアルタイム監視**: ClaudeのJSONLログファイルを監視し、新しいイベントが発生すると即座に表示
- **プロジェクト全体の監視**: スマートフィルタリングによる全プロジェクトのセッション監視
- **通知統合**: Claudeフック通知のリアルタイムキャプチャと表示
- **音声ナレーション**: VOICEVOXテキスト読み上げエンジンを使用したツールアクションの音声出力
- **AIナレーター**: OpenAIを使用したツールアクションの自然言語による説明
- **読みやすい出力**: タイムスタンプとフォーマットされたコンテンツによるイベント表示

## インストール

```bash
# リポジトリをクローン
git clone https://github.com/kazegusuri/claude-companion.git
cd claude-companion

# バイナリをビルド
make build
```

### Claudeフックの設定

Claude Companionはフックを通じてClaudeから通知イベントをキャプチャできます：

1. **通知スクリプトのインストール**:
   ```bash
   # 通知スクリプトを/usr/local/binにコピー
   sudo cp script/claude-notification.sh /usr/local/bin/
   sudo chmod +x /usr/local/bin/claude-notification.sh
   
   # 適切な権限でログファイルを作成
   sudo touch /var/log/claude-notification.log
   sudo chmod 666 /var/log/claude-notification.log
   ```

2. **Claudeフックの設定** (`~/.claude/settings.json`):
   ```json
   {
     "hooks": {
       "SessionStart": [
         {
           "matcher": "*",
           "hooks": [
             {
               "type": "command",
               "command": "/usr/local/bin/claude-notification.sh"
             }
           ]
         }
       ],
       "PreCompact": [
         {
           "matcher": "*",
           "hooks": [
             {
               "type": "command",
               "command": "/usr/local/bin/claude-notification.sh"
             }
           ]
         }
       ],
       "Notification": [
         {
           "matcher": "",
           "hooks": [
             {
               "type": "command",
               "command": "/usr/local/bin/claude-notification.sh"
             }
           ]
         }
       ]
     }
   }
   ```

## 使い方

### クイックスタート

```bash
# 音声とAIナレーション付きで全プロジェクトを監視（推奨）
./claude-companion --voice --ai

# 音声ナレーションなしで全プロジェクトを監視
./claude-companion

# 特定のプロジェクトを監視
./claude-companion -p myproject

# 特定のファイルを直接読み込み
./claude-companion -f /path/to/session.jsonl
```

### コマンドラインオプション

#### コアオプション
- `-p, --project`: 特定のプロジェクト名でフィルタリング
- `-s, --session`: 特定のセッション名でフィルタリング
- `-f, --file`: セッションファイルへの直接パス
- `--head`: tailingの代わりに最初から最後までファイル全体を読み込み
- `-d, --debug`: 詳細情報を含むデバッグモードを有効化

#### ナレーターオプション
- `--ai`: AIナレーターを使用（OpenAI APIキーが必要）
- `--openai-key`: OpenAI APIキー（OPENAI_API_KEY環境変数も使用可能）
- `--narrator-config`: カスタムナレーター設定ファイルへのパス

#### 音声オプション
- `--voice`: VOICEVOXを使用した音声出力を有効化
- `--voicevox-url`: VOICEVOXサーバーURL（デフォルト: http://localhost:50021）
- `--voice-speaker`: VOICEVOXスピーカーID（デフォルト: 1）

#### その他のオプション
- `--notification-log`: 通知ログファイルへのパス（デフォルト: /var/log/claude-notification.log）
- `--projects-root`: プロジェクトのルートディレクトリ（デフォルト: ~/.claude/projects）

## 動作モード

### 監視モード（デフォルト）

デフォルトでは、Claude Companionは`~/.claude/projects`以下の全プロジェクトを監視します。監視対象をフィルタリングできます：

```bash
# 全プロジェクトとセッションを監視
./claude-companion

# "myproject"のみを監視
./claude-companion -p myproject

# 全プロジェクトの"coding"という名前のセッションのみを監視
./claude-companion -s coding

# "myproject"の"coding"セッションのみを監視
./claude-companion -p myproject -s coding
```

ウォッチャーは自動的に：
- 新しいプロジェクトとセッションを検出
- ファイルの作成と削除を処理
- 複数のセッションウォッチャーを効率的に管理
- アイドル状態のウォッチャーを自動的にクリーンアップ

### 直接ファイルモード

特定のファイルを監視する場合：

```bash
# 特定のファイルをtail
./claude-companion -f /path/to/session.jsonl

# ファイル全体を読み込み
./claude-companion -f /path/to/session.jsonl --head
```

### 通知監視

ツールは`/var/log/claude-notification.log`が存在する場合、自動的に監視します：
- ファイルが存在しない場合は作成を待機
- 権限エラーを適切に処理
- 権限が付与されると監視を再開

**注意**: 通知監視にはClaudeフックの設定が必要です。通知スクリプトとClaudeの`settings.json`の設定方法については、上記の「Claudeフックの設定」セクションを参照してください。

## 音声ナレーション

### 前提条件

1. **VOICEVOXのインストール**（いずれか1つを選択）:
   - **Docker（最速の方法）**:
     ```bash
     docker run -d --rm -it -p '127.0.0.1:50021:50021' voicevox/voicevox_engine:cpu-latest
     ```
   - **手動インストール**:
     - ダウンロード: https://github.com/VOICEVOX/voicevox_engine
     - エンジンを実行: `./run`（Windowsでは`run.exe`）

2. **音声再生サポート**:
   - macOS: `afplay`（組み込み）
   - Linux: `aplay`または`paplay`
   - Windows: PowerShell（組み込み）

3. **AIナレーター（オプション）**:
   - `--ai`オプションを使用する場合は`OPENAI_API_KEY`環境変数の設定が必要
   - `--ai`オプションがない場合、英語の文章のナレーションがうまく動作しない可能性があります
   - `--ai`オプションを使用すると、テキストが日本語に翻訳されて読み上げられます

### 使用方法

```bash
# 音声ナレーションを有効化
./claude-companion --voice

# 特定のスピーカーを使用
./claude-companion --voice --voice-speaker 3

# より自然な説明のためのAIナレーター付き
./claude-companion --voice --ai
```

音声システムの機能：
- 重要なメッセージのための優先度ベースのキュー
- ノンブロッキング音声再生
- 適切なエラー処理
- 複数のスピーカーのサポート

## イベントタイプ

### 1. ユーザーイベント
```
[15:04:05] 👤 USER:
  💬 Hello, Claude!
```

### 2. アシスタントイベント
```
[15:04:06] 🤖 ASSISTANT (claude-3-sonnet):
  そのタスクをお手伝いします。
  
  💬 ファイル「main.go」を読み込みます
  📄 Reading: main.go
  
  💬 テストを実行します
  🏃 Running: make test
```

### 3. システムイベント
```
[15:04:07] ℹ️ SYSTEM [info]: ツールの実行が完了しました
[15:04:08] ⚠️ SYSTEM [warning]: レート制限に近づいています
```

### 4. 通知イベント
```
[15:04:09] 🔔 NOTIFICATION [SessionStart]:
  Project: myproject
  Session: coding-session
```

## ナレーター設定

カスタムナレーター設定ファイルを作成：

```json
{
  "rules": [
    {
      "pattern": "^git commit",
      "template": "{base} Gitにコミットします: {detail}"
    },
    {
      "tool": "Read",
      "template": "ファイル「{path}」を読み込みます"
    }
  ]
}
```

使用方法：
```bash
./claude-companion --narrator-config=/path/to/config.json
```

## 開発

[DEVELOPMENT.md](DEVELOPMENT.md)で開発手順を参照してください。

## ライセンス

MIT License