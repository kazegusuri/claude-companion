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

export interface ModelParameter {
  id: string;
  name: string;
  value: number;
  min: number;
  max: number;
  default: number;
}

export interface ModelMotion {
  group: string;
  index: number;
  name: string;
}

export interface ModelExpression {
  name: string;
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
  onModelInfoUpdate?: (info: {
    parameters: ModelParameter[];
    motions: ModelMotion[];
    expressions: ModelExpression[];
  }) => void;
  audioData?: string; // Base64 encoded audio data
  onAudioEnd?: () => void;
  modelRef?: React.MutableRefObject<Live2DModel | null>;
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
  onModelInfoUpdate,
  audioData,
  onAudioEnd,
  modelRef: externalModelRef,
}: Live2DModelViewerProps) {
  const canvasRef = useRef<HTMLDivElement>(null);
  const appRef = useRef<PIXI.Application | null>(null);
  const internalModelRef = useRef<Live2DModel | null>(null);
  const modelRef = externalModelRef || internalModelRef;
  const [modelId] = useState(() => `model-${Date.now()}`);
  const currentAudioUrlRef = useRef<string | null>(null);
  const isMouseInViewRef = useRef(false);
  const mouseEventCleanupRef = useRef<(() => void) | null>(null);

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

            // Extract model information
            const extractModelInfo = () => {
              const parameters: ModelParameter[] = [];
              const motions: ModelMotion[] = [];
              const expressions: ModelExpression[] = [];

              try {
                // Extract parameters
                if (model.internalModel && model.internalModel.coreModel) {
                  const coreModel = model.internalModel.coreModel as any;
                  const paramCount = coreModel.getParameterCount?.() || 0;

                  for (let i = 0; i < paramCount; i++) {
                    // Get parameter ID - this should return the actual parameter name
                    let paramId = "";
                    try {
                      // Try different methods to get the parameter ID
                      if (coreModel._parameterIds && coreModel._parameterIds[i]) {
                        paramId = coreModel._parameterIds[i];
                      } else if (coreModel.getParameterId) {
                        paramId = coreModel.getParameterId(i);
                      } else if (coreModel._model && coreModel._model._parameterIds) {
                        paramId = coreModel._model._parameterIds[i];
                      } else {
                        paramId = `param_${i}`;
                      }
                    } catch (e) {
                      paramId = `param_${i}`;
                    }

                    const value = coreModel.getParameterValueByIndex?.(i) || 0;
                    const min = coreModel.getParameterMinimumValueByIndex?.(i) || 0;
                    const max = coreModel.getParameterMaximumValueByIndex?.(i) || 1;
                    const defaultValue = coreModel.getParameterDefaultValueByIndex?.(i) || 0;

                    parameters.push({
                      id: String(i), // Use index as ID
                      name: paramId, // Use the actual parameter name
                      value,
                      min,
                      max,
                      default: defaultValue,
                    });
                  }
                }

                // Extract motions from motion manager
                if (model.internalModel && model.internalModel.motionManager) {
                  const motionManager = model.internalModel.motionManager as any;
                  const motionGroups = motionManager.motionGroups || {};

                  for (const [group, groupMotions] of Object.entries(motionGroups)) {
                    if (Array.isArray(groupMotions)) {
                      groupMotions.forEach((motion, index) => {
                        motions.push({
                          group,
                          index,
                          name: `${group}_${index}`,
                        });
                      });
                    }
                  }

                  // Also check definitions if available
                  if (motionManager.definitions) {
                    for (const [group, groupDefs] of Object.entries(motionManager.definitions)) {
                      if (Array.isArray(groupDefs)) {
                        groupDefs.forEach((def: any, index) => {
                          // Avoid duplicates
                          const exists = motions.find(
                            (m) => m.group === group && m.index === index,
                          );
                          if (!exists) {
                            motions.push({
                              group,
                              index,
                              name: def.file || `${group}_${index}`,
                            });
                          }
                        });
                      }
                    }
                  }
                }

                // Extract expressions from expression manager
                if (model.internalModel && model.internalModel.expressionManager) {
                  const expressionManager = model.internalModel.expressionManager as any;
                  const expressionDefinitions =
                    expressionManager.definitions || expressionManager.expressions || {};

                  for (const expName of Object.keys(expressionDefinitions)) {
                    expressions.push({ name: expName });
                  }

                  // Also check expression list if available
                  if (expressionManager.expressionList) {
                    expressionManager.expressionList.forEach((exp: any) => {
                      const name = exp.name || exp.Name || exp.file;
                      if (name && !expressions.find((e) => e.name === name)) {
                        expressions.push({ name });
                      }
                    });
                  }
                }
              } catch (error) {
                console.error("Error extracting model info:", error);
              }

              // Send info to parent
              if (onModelInfoUpdate) {
                onModelInfoUpdate({
                  parameters,
                  motions,
                  expressions,
                });
              }
            };

            // Extract model info
            extractModelInfo();

            // 正面を向かせる関数
            const lookForward = () => {
              if (!model) return;

              // まずFocusControllerを直接リセット
              if (model.internalModel && model.internalModel.focusController) {
                model.internalModel.focusController.focus(0, 0, true); // instant=true で即座に適用
              }

              // face_forward モーションがあるか確認
              try {
                // モーションを試行
                model.motion("face_forward");
              } catch {
                // face_forward モーションがない場合、パラメータを直接設定
                if (model.internalModel && model.internalModel.coreModel) {
                  const coreModel = model.internalModel.coreModel as any;

                  // 顔と目のX,Yパラメータを0に設定
                  const faceParams = [
                    "ParamAngleX",
                    "ParamAngleY",
                    "ParamAngleZ", // 顔の向き
                    "ParamEyeBallX",
                    "ParamEyeBallY", // 目の向き
                    "ParamBodyAngleX",
                    "ParamBodyAngleY",
                    "ParamBodyAngleZ", // 体の向き
                    "PARAM_ANGLE_X",
                    "PARAM_ANGLE_Y",
                    "PARAM_ANGLE_Z", // Cubism2用
                    "PARAM_EYE_BALL_X",
                    "PARAM_EYE_BALL_Y", // Cubism2用
                    "PARAM_BODY_ANGLE_X",
                    "PARAM_BODY_ANGLE_Y", // Cubism2用
                  ];

                  for (const paramName of faceParams) {
                    try {
                      const paramIndex = coreModel.getParameterIndex?.(paramName);
                      if (paramIndex >= 0) {
                        // すべて0にリセット
                        coreModel.setParameterValueByIndex?.(paramIndex, 0);
                      }
                    } catch (e) {
                      // パラメータが存在しない場合はスキップ
                    }
                  }

                  // 目の開きは1に設定
                  const eyeOpenParams = [
                    "ParamEyeLOpen",
                    "ParamEyeROpen",
                    "PARAM_EYE_L_OPEN",
                    "PARAM_EYE_R_OPEN",
                  ];
                  for (const paramName of eyeOpenParams) {
                    try {
                      const paramIndex = coreModel.getParameterIndex?.(paramName);
                      if (paramIndex >= 0) {
                        coreModel.setParameterValueByIndex?.(paramIndex, 1);
                      }
                    } catch (e) {
                      // パラメータが存在しない場合はスキップ
                    }
                  }
                }
              }
            };

            // マウス追跡とフォーカス制御を設定
            const handleMouseMove = (event: MouseEvent) => {
              if (!model || !canvasRef.current) return;

              const rect = canvasRef.current.getBoundingClientRect();
              const x = event.clientX - rect.left;
              const y = event.clientY - rect.top;

              // カーソルが画面内にあるかチェック
              if (x >= 0 && x <= rect.width && y >= 0 && y <= rect.height) {
                isMouseInViewRef.current = true;
                // モデルにフォーカスを設定
                model.focus(x, y);
              } else {
                // カーソルが画面外の場合
                if (isMouseInViewRef.current) {
                  isMouseInViewRef.current = false;
                  // 正面を向かせる
                  lookForward();
                }
              }
            };

            const handleMouseLeave = () => {
              if (!model || !canvasRef.current) return;

              isMouseInViewRef.current = false;

              // マウスが画面外に出たら正面を向かせる
              lookForward();
            };

            // イベントリスナーを追加
            window.addEventListener("mousemove", handleMouseMove);
            window.addEventListener("mouseleave", handleMouseLeave);
            document.addEventListener("mouseleave", handleMouseLeave);

            // クリーンアップ関数を保存
            mouseEventCleanupRef.current = () => {
              window.removeEventListener("mousemove", handleMouseMove);
              window.removeEventListener("mouseleave", handleMouseLeave);
              document.removeEventListener("mouseleave", handleMouseLeave);
            };

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

      // イベントリスナーを削除
      if (mouseEventCleanupRef.current) {
        mouseEventCleanupRef.current();
        mouseEventCleanupRef.current = null;
      }

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
          {...(specifiedWidth !== undefined && { specifiedWidth })}
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
        {...(specifiedWidth !== undefined && { specifiedWidth })}
      />
    </Box>
  );
}
