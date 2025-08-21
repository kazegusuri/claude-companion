# Claude Companion Web Frontend

## 開発環境と本番環境の分離

このプロジェクトでは、開発環境と本番環境を分離して運用できます。

### 開発環境（Development）

開発時は以下のコマンドで起動します。ファイルの変更が自動的に反映されます。

```bash
# ルートディレクトリから
bun run dev

# または webディレクトリから
cd web && bun run dev
```

- URL: http://localhost:5173
- ホットリロード有効
- ソースマップ有効
- デバッグ情報あり

### 本番環境（Production）

普段使い用の安定版は以下のコマンドで起動します。

```bash
# ルートディレクトリから
bun run start  # ビルドして起動
bun run serve  # ビルド済みを起動

# または webディレクトリから
cd web && bun run start
```

- URL: http://localhost:3001
- 最適化されたビルド
- ホットリロード無効（開発の影響を受けない）
- 高速動作

### 同時起動

開発環境と本番環境は異なるポートで動作するため、同時に起動できます：

1. ターミナル1: `bun run start` (本番環境 - ポート3001)
2. ターミナル2: `bun run dev` (開発環境 - ポート5173)

これにより、開発中のコード変更が本番環境に影響を与えることなく、安心して開発できます。

## 利用可能なスクリプト

| コマンド | 説明 | ポート |
|---------|------|--------|
| `bun run dev` | 開発サーバー起動 | 5173 |
| `bun run build` | プロダクションビルド | - |
| `bun run preview` | ビルド済みアプリのプレビュー | 4173 |
| `bun run serve` | プロダクションサーバー起動 | 3001 |
| `bun run start` | ビルド＆プロダクションサーバー起動 | 3001 |

## 環境変数

`.env`ファイルで環境変数を設定できます：

```env
# WebSocket URL (デフォルト: ws://localhost:8080/ws/audio)
VITE_WS_URL=ws://localhost:8080/ws/audio
```

開発環境と本番環境で異なる設定を使いたい場合は、`.env.development`と`.env.production`を使い分けることもできます。

## Docker を使った完全分離（オプション）

より完全な分離が必要な場合は、Dockerを使うこともできます。`Dockerfile`と`docker-compose.yml`が必要な場合はお知らせください。