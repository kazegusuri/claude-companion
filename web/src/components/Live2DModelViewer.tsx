import { useEffect, useRef, useState } from "react";
import * as PIXI from "pixi.js";
import { Live2DModel } from "pixi-live2d-display-lipsyncpatch/cubism4";
import { Box } from "@mantine/core";
import { SpeechBubble } from "./SpeechBubble";
import { Live2DModelStage } from "./Live2DModelStage";
import type { StageType } from "./Live2DModelStage";

// Window型を拡張してLive2D関連の型を追加
declare global {
  interface Window {
    PIXI: typeof PIXI;
  }
}

interface Live2DModelViewerProps {
  width?: number;
  height?: number;
  speechText?: string;
  isSpeaking?: boolean;
  bubbleSide?: "right" | "bottom" | "left" | "top";
  useCard?: boolean;
  cardTitle?: string;
  stageType?: StageType;
  bubbleMaxWidth?: number;
  specifiedWidth?: number;
  onModelLoaded?: (model: Live2DModel) => void;
  audioData?: string; // Base64 encoded audio data
  onAudioEnd?: () => void;
}

export function Live2DModelViewer({
  width = 360,
  height = 480,
  speechText = "",
  isSpeaking = false,
  bubbleSide = "bottom",
  useCard = false,
  cardTitle = "ASSISTANT",
  stageType = "gold-card",
  bubbleMaxWidth,
  specifiedWidth,
  onModelLoaded,
  audioData,
  onAudioEnd,
}: Live2DModelViewerProps) {
  const canvasRef = useRef<HTMLDivElement>(null);
  const appRef = useRef<PIXI.Application | null>(null);
  const modelRef = useRef<Live2DModel | null>(null);
  const [modelId] = useState(() => `model-${Date.now()}`);
  const currentAudioUrlRef = useRef<string | null>(null);

  // Audio playback using speak method
  useEffect(() => {
    if (modelRef.current && audioData) {
      const model = modelRef.current;

      // Convert base64 to blob URL
      try {
        // Stop current speaking if any
        model.stopSpeaking();

        // Clean up previous audio URL
        if (currentAudioUrlRef.current) {
          URL.revokeObjectURL(currentAudioUrlRef.current);
          currentAudioUrlRef.current = null;
        }

        // Convert base64 to blob
        const byteCharacters = atob(audioData.split(",")[1] || audioData);
        const byteNumbers = new Array(byteCharacters.length);
        for (let i = 0; i < byteCharacters.length; i++) {
          byteNumbers[i] = byteCharacters.charCodeAt(i);
        }
        const byteArray = new Uint8Array(byteNumbers);
        const blob = new Blob([byteArray], { type: "audio/wav" });
        const audioUrl = URL.createObjectURL(blob);
        currentAudioUrlRef.current = audioUrl;

        // Use speak method for lip sync
        model.speak(audioUrl, {
          volume: 1.0,
          onFinish: () => {
            if (currentAudioUrlRef.current) {
              URL.revokeObjectURL(currentAudioUrlRef.current);
              currentAudioUrlRef.current = null;
            }
            onAudioEnd?.();
          },
          onError: (error) => {
            console.error("Error playing audio with speak:", error);
            if (currentAudioUrlRef.current) {
              URL.revokeObjectURL(currentAudioUrlRef.current);
              currentAudioUrlRef.current = null;
            }
            onAudioEnd?.();
          },
          crossOrigin: "anonymous",
        });
      } catch (error) {
        console.error("Error converting audio data:", error);
        onAudioEnd?.();
      }
    }

    // Cleanup on unmount or when audioData changes
    return () => {
      if (modelRef.current) {
        modelRef.current.stopSpeaking();
      }
      if (currentAudioUrlRef.current) {
        URL.revokeObjectURL(currentAudioUrlRef.current);
        currentAudioUrlRef.current = null;
      }
    };
  }, [audioData, onAudioEnd]);

  useEffect(() => {
    if (!canvasRef.current) return;

    // PIXIをwindowオブジェクトに設定（pixi-live2d-displayが必要とするため）
    window.PIXI = PIXI;

    let app: PIXI.Application | null = null;
    let model: Live2DModel | null = null;

    // Wait for container to have dimensions
    const initTimeout = setTimeout(async () => {
      if (!canvasRef.current) return;

      try {
        // Get actual container dimensions
        const containerWidth = width || canvasRef.current.clientWidth || 360;
        const containerHeight = height || canvasRef.current.clientHeight || 480;

        // PixiJSアプリケーションを作成 (PIXI v7用)
        app = new PIXI.Application({
          width: containerWidth,
          height: containerHeight,
          backgroundColor: 0x000000,
          backgroundAlpha: 0,
          antialias: true,
        });

        // canvasをDOMに追加
        if (canvasRef.current) {
          canvasRef.current.appendChild(app.view as HTMLCanvasElement);
        }

        appRef.current = app;

        // サンプルモデルを読み込む（モデルファイルが存在する場合）
        try {
          const modelName = import.meta.env.VITE_LIVE2D_MODEL_NAME || "default";
          const modelPath = `/live2d/models/${modelName}/${modelName}.model3.json`;

          // モデルが存在するかチェック（404エラーを避けるため）
          const response = await fetch(modelPath, { method: "HEAD" });

          if (response.ok) {
            // Live2Dモデルを読み込み
            model = await Live2DModel.from(modelPath);

            // モデルをステージに追加（サイズ計算のため先に追加）
            app.stage.addChild(model as any);

            // モデルのバウンディングボックスを取得
            const bounds = model.getLocalBounds();
            const modelWidth = bounds.width;
            const modelHeight = bounds.height;

            // アスペクト比を保持しながらスケールを計算（コンテナの85%に収める）
            const targetSize = 0.85; // 85%に調整
            const scaleX = (containerWidth * targetSize) / modelWidth;
            const scaleY = (containerHeight * targetSize) / modelHeight;
            const scale = Math.min(scaleX, scaleY); // アスペクト比を保持

            // スケールを適用
            model.scale.set(scale);

            // スケール後のサイズを計算
            const scaledWidth = modelWidth * scale;
            const scaledHeight = modelHeight * scale;

            // モデルの中心をキャンバスの中心に配置（少し下寄りに調整）
            model.x = (containerWidth - scaledWidth) / 2;
            model.y = (containerHeight - scaledHeight) / 2 + containerHeight * 0.05; // 5%下にオフセット

            // インタラクション設定
            model.on("hit", (hitAreas: string[]) => {
              // タップされた部位に応じてモーションを再生
              if (hitAreas.includes("Body")) {
                model?.motion("Tap");
              }
            });

            // Store model reference
            modelRef.current = model;

            // Call callback if provided
            if (onModelLoaded) {
              onModelLoaded(model);
            }

            // アイドルモーションを開始
            model.motion("Idle");
          }
        } catch (error) {
          // モデルが見つからない場合は何も表示しない（エラーは出力しない）
        }
      } catch (error) {
        console.error("Failed to initialize Live2D viewer:", error);
      }
    }, 100); // Wait 100ms for container to be ready

    // クリーンアップ
    return () => {
      clearTimeout(initTimeout);
      if (model) {
        model.destroy();
      }
      if (modelRef.current) {
        modelRef.current = null;
      }
      if (appRef.current) {
        appRef.current.destroy(true);
        appRef.current = null;
      }
    };
  }, [width, height, onModelLoaded]);

  const modelContent = (
    <Box
      ref={canvasRef}
      id={modelId}
      className="tcg-art"
      style={{
        position: "absolute",
        inset: 0,
        width: "100%",
        height: "100%",
        display: "flex",
        justifyContent: "center",
        alignItems: "center",
        zIndex: 2, // 背景の上に配置
      }}
    />
  );

  if (useCard) {
    return (
      <Box style={{ position: "relative", width: "100%", height: "100%" }}>
        <Box
          style={{
            display: "flex",
            justifyContent: "center",
            alignItems: "center",
            width: "100%",
            height: "100%",
          }}
        >
          <Live2DModelStage type={stageType} title={cardTitle}>
            {modelContent}
          </Live2DModelStage>
        </Box>
        <SpeechBubble
          text={speechText}
          visible={isSpeaking}
          anchorSelector={`.live2d-stage`}
          side={bubbleSide}
          withWave={true}
          typewriter={true}
          maxWidth={bubbleMaxWidth || 600}
          isMobile={window.location.pathname === "/mobile"}
          specifiedWidth={specifiedWidth}
          style={{
            zIndex: 1000,
          }}
        />
      </Box>
    );
  }

  return (
    <Box style={{ position: "relative", width: "100%", height: "100%" }}>
      {modelContent}
      <SpeechBubble
        text={speechText}
        visible={isSpeaking}
        anchorSelector={`#${modelId}`}
        side={bubbleSide}
        withWave={true}
        typewriter={true}
        maxWidth={bubbleMaxWidth || 600}
        isMobile={window.location.pathname === "/mobile"}
        specifiedWidth={specifiedWidth}
      />
    </Box>
  );
}
