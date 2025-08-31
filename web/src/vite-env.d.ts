/// <reference types="vite/client" />

// 環境変数の型定義
interface ImportMetaEnv {
  readonly VITE_API_URL: string;
  readonly VITE_ENV: "development" | "production" | "test";
  readonly VITE_WS_URL?: string; // WebSocket URL
  readonly VITE_LIVE2D_MODEL_NAME?: string; // Live2Dモデル名（省略時は「default」を使用）
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
