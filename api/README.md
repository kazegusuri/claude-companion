# Claude Companion API Specification

このディレクトリには、Claude Companion APIの仕様をTypeSpecで定義しています。

## TypeSpecとは

TypeSpecは、Microsoftが開発したAPI定義言語で、TypeScriptに似た構文でAPIを記述し、OpenAPI、JSON Schema、その他の形式に変換できます。

## ファイル構造

```
api/
├── main.tsp            # API定義のメインファイル
├── tspconfig.yaml      # TypeSpec設定ファイル
├── package.json        # Node.js依存関係
└── tsp-output/         # 生成された出力
    └── @typespec/
        ├── openapi3/   # OpenAPI仕様
        └── json-schema/ # JSON Schema
```

## セットアップ

```bash
# 依存関係のインストール
npm install

# TypeSpecのコンパイル
npx tsp compile .
```

## 使用方法

### API定義の編集

`main.tsp`ファイルを編集してAPI定義を変更します。

### OpenAPIとTypeScriptの型生成

webディレクトリから以下のコマンドを実行：

```bash
# TypeSpecをコンパイルしてOpenAPIを生成し、TypeScriptの型を生成
bun run tsp:generate
```

これにより：
1. TypeSpecがコンパイルされ、OpenAPI仕様が生成されます
2. OpenAPIからTypeScriptの型定義が`web/src/types/api.ts`に生成されます

### 生成されるファイル

- **OpenAPI仕様**: `tsp-output/@typespec/openapi3/openapi.yaml`
  - Swagger UIやPostmanで利用可能
  - APIドキュメント生成に使用

- **TypeScript型定義**: `web/src/types/api.ts`
  - フロントエンドで型安全なAPI呼び出しが可能
  - 自動生成されるため、手動編集は不要

## API定義の構造

### モデル定義

```typespec
model Agent {
  pid: int32;
  sessionId: string;
  projectDir: string;
  @format("date-time")
  createdAt: string;
  @format("date-time")
  updatedAt: string;
}
```

### インターフェース定義

```typespec
@route("/api/agents")
@tag("Agents")
interface Agents {
  @get
  list(): AgentListResponse;
  
  @get
  @route("{pid}")
  read(@path pid: int32): Agent | ErrorResponse;
  
  @delete
  @route("{pid}")
  delete(@path pid: int32): void | ErrorResponse;
}
```

## 利点

1. **単一の真実の源**: API仕様を一箇所で管理
2. **型安全性**: TypeScriptの型が自動生成される
3. **標準準拠**: OpenAPI 3.0仕様を生成
4. **開発効率**: APIの変更が自動的にフロントエンドの型に反映される

## 拡張方法

新しいAPIエンドポイントを追加する場合：

1. `main.tsp`に新しいモデルやインターフェースを追加
2. `bun run tsp:generate`を実行
3. 生成された型を使用してフロントエンドを実装

## 参考リンク

- [TypeSpec公式サイト](https://typespec.io/)
- [TypeSpec GitHub](https://github.com/microsoft/typespec)
- [OpenAPI Specification](https://swagger.io/specification/)