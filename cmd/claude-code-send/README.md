# claude-code-send

Emacs vterm と連携して Claude Code セッションを制御するためのコマンドラインツール。

## 概要

このツールは標準入力から JSON を受け取り、指定されたコマンドに基づいて Emacs の vterm バッファにキー入力を送信します。

## 使用方法

```bash
echo '{"cwd": "/path/to/project", "sessionId": "session-id", "command": "proceed"}' | claude-code-send
```

## 入力フォーマット

```json
{
  "cwd": "/path/to/project",
  "sessionId": "unique-session-id",
  "command": "proceed|stop|send",
  "message": "optional message for send command"
}
```

## コマンド

- **proceed**: Enter キーを送信（処理を続行）
- **stop**: Escape キーを送信（処理を中断）
- **send**: メッセージを送信して Enter キーを押す

## 出力フォーマット

成功時:
```json
{
  "success": true,
  "message": "Command executed successfully",
  "sessionId": "session-id"
}
```

エラー時:
```json
{
  "success": false,
  "error": "Error description",
  "sessionId": "session-id"
}
```

## 動作原理

1. CWD からベースディレクトリ名を抽出
2. `*claude-code[ディレクトリ名]*` という名前の vterm バッファを対象に操作
3. `emacsclient` を使用して Emacs Lisp コマンドを実行

## ビルド

```bash
go build -o claude-code-send main.go
```

## 依存関係

- Emacs が起動していること
- emacsclient が使用可能であること
- 対象の vterm バッファが存在すること