import { useEffect, useRef } from "react";
import * as PIXI from "pixi.js";
import { Live2DModel } from "pixi-live2d-display-lipsyncpatch/cubism4";

// Window型を拡張してLive2D関連の型を追加
declare global {
  interface Window {
    PIXI: typeof PIXI;
  }
}

export function Live2DViewer() {
  const canvasRef = useRef<HTMLDivElement>(null);
  const appRef = useRef<PIXI.Application | null>(null);

  useEffect(() => {
    if (!canvasRef.current) return;

    // PIXIをwindowオブジェクトに設定（pixi-live2d-displayが必要とするため）
    window.PIXI = PIXI;

    let app: PIXI.Application | null = null;
    let model: Live2DModel | null = null;

    (async () => {
      try {
        // PixiJSアプリケーションを作成 (PIXI v7用)
        app = new PIXI.Application({
          width: 1000,
          height: 800,
          backgroundColor: 0xffffff,
          antialias: true,
        });

        // canvasをDOMに追加
        if (canvasRef.current) {
          canvasRef.current.appendChild(app.view as HTMLCanvasElement);
        }

        appRef.current = app;

        // 背景を作成 (PIXI v7用)
        const background = new PIXI.Graphics();
        background.beginFill(0xf0f0f0);
        background.drawRect(0, 0, 1000, 800);
        background.endFill();
        app.stage.addChild(background);

        // 説明テキスト (PIXI v7用)
        const infoText = new PIXI.Text("Live2D Model Viewer", {
          fontFamily: "Arial",
          fontSize: 24,
          fill: 0x333333,
        });
        infoText.x = 500;
        infoText.y = 30;
        infoText.anchor.set(0.5);
        app.stage.addChild(infoText);

        // Live2Dモデル読み込みの説明 (PIXI v7用)
        const instructionText = new PIXI.Text(
          "Place your Live2D model files in:\n/public/live2d/models/[model_name]/\n\nRequired files:\n- model3.json (or model.json)\n- *.moc3 file\n- texture files\n- *.physics3.json (optional)\n- *.motion3.json files (optional)",
          {
            fontFamily: "Arial",
            fontSize: 14,
            fill: 0x666666,
            align: "center",
            lineHeight: 20,
          },
        );
        instructionText.x = 500;
        instructionText.y = 400;
        instructionText.anchor.set(0.5);
        app.stage.addChild(instructionText);

        // サンプルモデルを読み込む（モデルファイルが存在する場合）
        // 環境変数からモデル名を取得（デフォルト: default）
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

            // キャンバスサイズ
            const canvasWidth = 1000;
            const canvasHeight = 800;

            // アスペクト比を保持しながらスケールを計算（キャンバスの70%に収める）
            const targetSize = 0.7; // 70%
            const scaleX = (canvasWidth * targetSize) / modelWidth;
            const scaleY = (canvasHeight * targetSize) / modelHeight;
            const scale = Math.min(scaleX, scaleY); // アスペクト比を保持

            // スケールを適用
            model.scale.set(scale);

            // スケール後のサイズを計算
            const scaledWidth = modelWidth * scale;
            const scaledHeight = modelHeight * scale;

            // モデルの中心をキャンバスの中心に配置
            // Live2Dモデルは左上が原点なので、中心に配置するために調整
            model.x = (canvasWidth - scaledWidth) / 2;
            model.y = (canvasHeight - scaledHeight) / 2;

            console.log("Model dimensions:", {
              originalWidth: modelWidth,
              originalHeight: modelHeight,
              scaledWidth,
              scaledHeight,
              scale,
              x: model.x,
              y: model.y,
            });

            // インタラクション設定
            model.on("hit", (hitAreas: string[]) => {
              console.log("Hit areas:", hitAreas);
              // タップされた部位に応じてモーションを再生
              if (hitAreas.includes("Body")) {
                model?.motion("Tap");
              }
            });

            // 説明テキストを更新
            instructionText.text = "Model loaded successfully!\nClick on the model to interact.";
          }
        } catch (error) {
          console.log("No model found or error loading model:", error);
          // モデルが見つからない場合は説明テキストのままにする
        }

        // デバッグ用のボタン（サンプルモーション再生） (PIXI v7用)
        const button = new PIXI.Graphics();
        button.beginFill(0x4caf50);
        button.drawRect(0, 0, 200, 40);
        button.endFill();
        button.x = 400;
        button.y = 720;
        button.eventMode = 'static';
        button.cursor = 'pointer';

        const buttonText = new PIXI.Text("Play Motion", {
          fontFamily: "Arial",
          fontSize: 16,
          fill: 0xffffff,
        });
        buttonText.x = 500;
        buttonText.y = 740;
        buttonText.anchor.set(0.5);

        button.on("pointerdown", () => {
          if (model) {
            // ランダムなモーションを再生
            model.motion("Idle");
          }
        });

        app.stage.addChild(button);
        app.stage.addChild(buttonText);
      } catch (error) {
        console.error("Failed to initialize Live2D viewer:", error);
      }
    })();

    // クリーンアップ
    return () => {
      if (model) {
        model.destroy();
      }
      if (appRef.current) {
        appRef.current.destroy(true);
        appRef.current = null;
      }
    };
  }, []);

  return (
    <div
      ref={canvasRef}
      style={{
        display: "flex",
        justifyContent: "center",
        padding: "20px",
        backgroundColor: "#f5f5f5",
        borderRadius: "8px",
        boxShadow: "0 2px 4px rgba(0,0,0,0.1)",
      }}
    />
  );
}
