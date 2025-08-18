import React from "react";
import GoldFrameStage from "./GoldFrameStage";

export type StageType = "gold-card" | "simple" | "minimal";

interface Live2DModelStageProps {
  type?: StageType;
  title?: string;
  children: React.ReactNode;
  className?: string;
}

/**
 * Live2Dモデルを表示するステージコンポーネント
 * 固定サイズ440x540のステージ内にcanvas(320x480)を配置
 * typeプロパティで異なるフレームデザインに切り替え可能
 */
export function Live2DModelStage({
  type = "gold-card",
  title = "ASSISTANT",
  children,
  className = "",
}: Live2DModelStageProps) {
  // ステージの固定サイズ
  const STAGE_WIDTH = 440;
  const STAGE_HEIGHT = 540;

  // canvasの固定サイズ
  const CANVAS_WIDTH = 360;
  const CANVAS_HEIGHT = 480;

  // typeに応じて異なるステージを返す
  switch (type) {
    case "gold-card":
      // GoldCardFrameを使用した装飾的なステージ
      return (
        <div
          className={`live2d-stage ${className}`}
          style={{
            width: STAGE_WIDTH,
            height: STAGE_HEIGHT,
            position: "relative",
          }}
        >
          <GoldFrameStage width={STAGE_WIDTH} title={title}>
            {children}
          </GoldFrameStage>
        </div>
      );

    case "simple":
      // シンプルなボーダー付きステージ
      return (
        <div
          className={`live2d-stage ${className}`}
          style={{
            width: STAGE_WIDTH,
            height: STAGE_HEIGHT,
            position: "relative",
            backgroundColor: "#1a1a1a",
            borderRadius: "12px",
            border: "2px solid #444",
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            padding: "20px",
          }}
        >
          {title && (
            <div
              style={{
                color: "#fff",
                fontSize: "18px",
                fontWeight: "bold",
                marginBottom: "10px",
                textTransform: "uppercase",
                letterSpacing: "2px",
              }}
            >
              {title}
            </div>
          )}
          <div
            style={{
              width: CANVAS_WIDTH,
              height: CANVAS_HEIGHT,
              position: "relative",
              backgroundColor: "#000",
              borderRadius: "8px",
              overflow: "hidden",
            }}
          >
            {children}
          </div>
        </div>
      );

    case "minimal":
      // 最小限のステージ（フレームなし）
      return (
        <div
          className={`live2d-stage ${className}`}
          style={{
            width: STAGE_WIDTH,
            height: STAGE_HEIGHT,
            position: "relative",
            display: "flex",
            justifyContent: "center",
            alignItems: "center",
            backgroundColor: "transparent",
          }}
        >
          <div
            style={{
              width: CANVAS_WIDTH,
              height: CANVAS_HEIGHT,
              position: "relative",
            }}
          >
            {children}
          </div>
        </div>
      );

    default:
      return null;
  }
}
