import path from "node:path";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],

  // 開発サーバー設定
  server: {
    port: 3000, // ポート番号を固定
    strictPort: true, // ポートが使用中の場合はエラーにする
    host: true, // ネットワーク上の他のデバイスからアクセス可能にする
  },

  // プレビューサーバー設定
  preview: {
    port: 5550,
    strictPort: true,
  },

  // パスエイリアス設定
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      "@components": path.resolve(__dirname, "./src/components"),
      "@assets": path.resolve(__dirname, "./src/assets"),
      "@hooks": path.resolve(__dirname, "./src/hooks"),
      "@utils": path.resolve(__dirname, "./src/utils"),
    },
  },

  // ビルド最適化
  build: {
    target: "esnext", // 最新のブラウザをターゲット
    minify: "esbuild", // esbuildを使用して高速化
    sourcemap: true, // ソースマップを生成（デバッグ用）
    rollupOptions: {
      output: {
        // チャンク分割の最適化
        manualChunks: {
          react: ["react", "react-dom"],
        },
      },
    },
  },

  // 最適化設定
  optimizeDeps: {
    include: ["react", "react-dom"], // 事前バンドル対象を明示
  },
});
