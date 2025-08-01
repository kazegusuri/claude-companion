# Example Session Files

This directory contains example Claude session files for testing and demonstration purposes.

## sample_session.jsonl

A minimal example session demonstrating basic Claude interactions with a Go project:

1. User greeting and request for Go project information
2. Assistant attempts to read README.md (not found)
3. Lists directory contents
4. Reads main.go file
5. Executes the Go program
6. Provides summary of the project

### Usage

```bash
# Basic usage (no voice)
./claude-companion -file example/sample_session.jsonl -full

# With voice output (requires VOICEVOX running)
./claude-companion -file example/sample_session.jsonl -full -voice
```

### Voice Output Examples

When running with `-voice` option, you'll hear:
- 「ドキュメント「README.md」を読み込みます」
- 「現在のディレクトリの内容を確認します」
- 「Goファイル「main.go」を読み込みます」
- 「Goプログラムを実行します」