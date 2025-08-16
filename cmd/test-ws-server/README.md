# WebSocket Test Server

WebSocket音声システムのテスト用サーバーです。フロントエンドの開発やE2Eテストで使用します。

## 起動方法

```bash
go run cmd/test-ws-server/main.go
```

サーバーは以下のポートで起動します：
- WebSocket: `ws://localhost:8080/ws/audio`
- HTTP API: `http://localhost:8080`

## API エンドポイント

### 1. テキストメッセージ送信

テキストのみのメッセージを送信します。

```bash
curl -X POST http://localhost:8080/api/send/text \
  -H "Content-Type: application/json" \
  -d '{
    "text": "テストメッセージです",
    "eventType": "message",
    "toolName": "CurlTest"
  }'
```

**パラメータ:**
- `text` (必須): 送信するテキスト
- `eventType` (任意): イベントタイプ（例: "message", "tool", "error"）
- `toolName` (任意): ツール名

### 2. オーディオメッセージ送信

音声データ付きのメッセージを送信します。

```bash
curl -X POST http://localhost:8080/api/send/audio \
  -H "Content-Type: application/json" \
  -d '{
    "text": "音声付きメッセージです",
    "eventType": "tool",
    "toolName": "AudioTest",
    "sampleRate": 44100,
    "duration": 0.5
  }'
```

**パラメータ:**
- `text` (必須): 送信するテキスト
- `audioData` (任意): Base64エンコードされたWAVデータ（省略時はダミー音声を生成）
- `eventType` (任意): イベントタイプ
- `toolName` (任意): ツール名
- `sampleRate` (任意): サンプルレート（デフォルト: 44100）
- `duration` (任意): 音声の長さ（秒）

### 3. テストメッセージ送信（簡易版）

プリセットされたテストメッセージを送信します。

```bash
curl -X POST http://localhost:8080/api/send/test
```

呼び出すたびに異なるメッセージが送信されます：
- "テストメッセージ1: ファイルを読み込んでいます"
- "テストメッセージ2: ビルドを実行中です"
- "テストメッセージ3: テストが完了しました"
- など

**音声データについて:**
- カレントディレクトリに `sample.wav` ファイルがある場合は、それを使用
- `sample.wav` が見つからない場合は、ダミーのサイレントWAVを生成

サンプル音声を使用する場合：
```bash
# sample.wavをカレントディレクトリに配置
cp /path/to/your/audio.wav sample.wav

# テストサーバーを起動
go run cmd/test-ws-server/main.go

# テストメッセージを送信（sample.wavが使用される）
curl -X POST http://localhost:8080/api/send/test
```

### 4. ヘルスチェック

サーバーの状態と接続中のクライアント数を確認します。

```bash
curl http://localhost:8080/health
```

**レスポンス例:**
```json
{
  "status": "ok",
  "clients": 2
}
```

## 実際のWAVファイルを送信

VOICEVOXなどで生成した実際のWAVファイルを送信する場合：

```bash
# WAVファイルをBase64エンコードして送信
WAV_BASE64=$(base64 -w0 < your_audio.wav)
curl -X POST http://localhost:8080/api/send/audio \
  -H "Content-Type: application/json" \
  -d "{
    \"text\": \"実際の音声ファイル\",
    \"audioData\": \"$WAV_BASE64\",
    \"eventType\": \"audio\",
    \"sampleRate\": 44100
  }"
```

## 連続メッセージ送信

複数のメッセージを連続して送信する例：

```bash
# 3つのメッセージを連続送信
for i in {1..3}; do
  curl -X POST http://localhost:8080/api/send/text \
    -H "Content-Type: application/json" \
    -d "{\"text\": \"メッセージ $i\"}"
  sleep 0.5
done
```

## E2Eテストでの使用

WebSocket音声システムのE2Eテストを実行する場合：

```bash
# 1. テストサーバーを起動
go run cmd/test-ws-server/main.go

# 2. 別ターミナルでWebアプリケーションの開発サーバーを起動
cd web && bun run dev

# 3. さらに別ターミナルでE2Eテストを実行
cd web && bun run test:e2e
```

## WebSocketメッセージフォーマット

サーバーから送信されるWebSocketメッセージのフォーマット：

```json
{
  "type": "audio",
  "id": "uuid-string",
  "text": "メッセージテキスト",
  "audioData": "base64-encoded-wav-data",
  "priority": 5,
  "timestamp": "2024-01-01T00:00:00Z",
  "metadata": {
    "eventType": "tool",
    "toolName": "TestTool",
    "sampleRate": 44100,
    "duration": 0.5
  }
}
```

## トラブルシューティング

### ポート8080が使用中の場合

他のプロセスがポート8080を使用している場合は、そのプロセスを終了するか、ソースコードでポートを変更してください。

```bash
# ポート8080を使用しているプロセスを確認
lsof -i :8080

# プロセスを終了
kill -9 <PID>
```

### CORSエラーが発生する場合

フロントエンドのURLが`localhost:3000`以外の場合は、`cmd/test-ws-server/main.go`のCORS設定を更新してください：

```go
c := cors.New(cors.Options{
    AllowedOrigins: []string{"*"}, // すべてのオリジンを許可（開発用）
    // ...
})
```