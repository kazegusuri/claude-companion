import { Box } from "@mantine/core";
import * as PIXI from "pixi.js";
import * as PixiLive2D from "pixi-live2d-display-lipsyncpatch/cubism4";
import { Live2DModel } from "pixi-live2d-display-lipsyncpatch/cubism4";
import { useEffect, useRef, useState } from "react";
import { AudioPlayer } from "../services/AudioPlayer";
import {
  findAndPatchSoundManager,
  resumeSharedAudioContext,
} from "../utils/live2d/audioContextPatch";
import type { StageType } from "./Live2DModelStage";
import { Live2DModelStage } from "./Live2DModelStage";
import { SpeechBubble } from "./SpeechBubble";

// グループ名の定数
const GROUP_NAME_FOCUS = "Focus";

// モーション名の定数
const MOTION_IDLE = "Idle";
const MOTION_TAP = "Tap";
const MOTION_FACE_FORWARD = "FaceForward";

// デフォルトのフォーカスパラメータ
const DEFAULT_FOCUS_PARAMS = [
  "ParamAngleX",
  "ParamAngleY",
  "ParamAngleZ",
  "ParamEyeBallX",
  "ParamEyeBallY",
  "ParamBodyAngleX",
  "ParamBodyAngleY",
  "ParamBodyAngleZ",
];

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

// Live2D Model3 Group定義
interface Live2DModelGroup {
  Name?: string;
  name?: string;
  Ids?: string[];
}

// Live2D内部モデルの型定義
interface Live2DCoreModel {
  getParameterCount(): number;
  getParameterId(index: number): string;
  getParameterValueByIndex(index: number): number;
  getParameterMinimumValueByIndex(index: number): number;
  getParameterMaximumValueByIndex(index: number): number;
  getParameterDefaultValueByIndex(index: number): number;
  setParameterValueByIndex(index: number, value: number): void;
  getParameterIndex(id: string): number;
  _parameterIds?: string[];
  _model?: {
    getParameterCount(): number;
    getParameterId(index: number): string;
    getParameterValueByIndex(index: number): number;
    getParameterMinimumValueByIndex(index: number): number;
    getParameterMaximumValueByIndex(index: number): number;
    getParameterDefaultValueByIndex(index: number): number;
    setParameterValueByIndex(index: number, value: number): void;
    getParameterIndex(id: string): number;
    _parameterIds?: string[];
  };
}

// Motion Manager型定義
interface MotionManager {
  motionGroups?: Record<string, unknown[]>;
  definitions?: Record<string, unknown[]>;
}

// Expression Manager型定義
interface ExpressionManager {
  definitions?: Record<string, unknown>;
  expressions?: Record<string, unknown>;
  expressionList?: Array<{
    name?: string;
    Name?: string;
    file?: string;
  }>;
}

interface ExtendedInternalModel {
  coreModel?: Live2DCoreModel;
  motionManager?: MotionManager;
  expressionManager?: ExpressionManager;
  settings?: {
    groups?: Live2DModelGroup[];
  };
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
  const audioPlayerRef = useRef<AudioPlayer | null>(null);
  const lipSyncIntervalRef = useRef<number | null>(null);

  // Fallback audio playback using AudioPlayer
  const playWithAudioPlayer = async (base64Data: string) => {
    try {
      if (!audioPlayerRef.current) {
        audioPlayerRef.current = new AudioPlayer();
      }

      // Initialize AudioPlayer if needed
      if (!audioPlayerRef.current.isContextInitialized()) {
        await audioPlayerRef.current.ensureInitialized();
      }

      // Clear any existing lip sync interval
      if (lipSyncIntervalRef.current) {
        clearInterval(lipSyncIntervalRef.current);
        lipSyncIntervalRef.current = null;
      }

      // Play audio with lip sync simulation
      await audioPlayerRef.current.playBase64Audio(base64Data, {
        onStart: () => {
          // Start lip sync simulation for PWA
          if (modelRef.current?.internalModel) {
            let mouthValue = 0;
            let targetValue = 0;

            lipSyncIntervalRef.current = window.setInterval(() => {
              // Generate random mouth movements during playback
              targetValue = Math.random() * 0.8 + 0.2; // 0.2 to 1.0 range
              mouthValue += (targetValue - mouthValue) * 0.3; // Smooth transition

              // Set mouth parameter directly
              try {
                const coreModel = modelRef.current?.internalModel?.coreModel;
                if (coreModel) {
                  // Common mouth parameter IDs
                  const mouthParams = ["ParamMouthOpenY", "PARAM_MOUTH_OPEN_Y", "ParamMouthOpen"];
                  for (const paramName of mouthParams) {
                    const paramIndex = (coreModel as Live2DCoreModel).getParameterIndex?.(
                      paramName,
                    );
                    if (paramIndex >= 0) {
                      (coreModel as Live2DCoreModel).setParameterValueByIndex?.(
                        paramIndex,
                        mouthValue,
                      );
                      break;
                    }
                  }
                }
              } catch (_e) {
                // Ignore parameter errors
              }
            }, 50); // Update every 50ms
          }
        },
        onEnd: () => {
          // Stop lip sync
          if (lipSyncIntervalRef.current) {
            clearInterval(lipSyncIntervalRef.current);
            lipSyncIntervalRef.current = null;
          }

          // Reset mouth to closed
          try {
            const coreModel = modelRef.current?.internalModel?.coreModel;
            if (coreModel) {
              const mouthParams = ["ParamMouthOpenY", "PARAM_MOUTH_OPEN_Y", "ParamMouthOpen"];
              for (const paramName of mouthParams) {
                const paramIndex = (coreModel as Live2DCoreModel).getParameterIndex?.(paramName);
                if (paramIndex >= 0) {
                  (coreModel as Live2DCoreModel).setParameterValueByIndex?.(paramIndex, 0);
                  break;
                }
              }
            }
          } catch (_e) {
            // Ignore parameter errors
          }

          onAudioEnd?.();
        },
        onError: (error) => {
          console.error("AudioPlayer: Error during playback", error);
          if (lipSyncIntervalRef.current) {
            clearInterval(lipSyncIntervalRef.current);
            lipSyncIntervalRef.current = null;
          }
          onAudioEnd?.();
        },
        onVolumeUpdate: (volume) => {
          // Use volume data for more accurate lip sync if available
          if (modelRef.current?.internalModel?.coreModel) {
            try {
              const coreModel = modelRef.current.internalModel.coreModel;
              const mouthParams = ["ParamMouthOpenY", "PARAM_MOUTH_OPEN_Y", "ParamMouthOpen"];
              for (const paramName of mouthParams) {
                const paramIndex = (coreModel as Live2DCoreModel).getParameterIndex?.(paramName);
                if (paramIndex >= 0) {
                  (coreModel as Live2DCoreModel).setParameterValueByIndex?.(paramIndex, volume);
                  break;
                }
              }
            } catch (_e) {
              // Ignore parameter errors
            }
          }
        },
      });
    } catch (error) {
      console.error("AudioPlayer: Failed to play audio", error);
      onAudioEnd?.();
    }
  };

  // Audio playback using speak method with fallback
  useEffect(() => {
    const model = modelRef.current;
    if (model && audioData) {
      // Convert base64 to blob URL
      try {
        // PWA対策: 音声再生前に共有AudioContextをresume
        resumeSharedAudioContext();

        // Stop current speaking if any
        model.stopSpeaking();

        // Stop AudioPlayer if playing
        audioPlayerRef.current?.stop();

        // Clear lip sync interval
        if (lipSyncIntervalRef.current) {
          clearInterval(lipSyncIntervalRef.current);
          lipSyncIntervalRef.current = null;
        }

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
          onError: (error: unknown) => {
            console.error("model.speak failed, using AudioPlayer fallback:", error);

            // Clean up blob URL
            if (currentAudioUrlRef.current) {
              URL.revokeObjectURL(currentAudioUrlRef.current);
              currentAudioUrlRef.current = null;
            }

            // Fallback to AudioPlayer
            playWithAudioPlayer(audioData);
          },
          crossOrigin: "anonymous",
        });
      } catch (error) {
        console.error("Error setting up audio playback:", error);
        // Try fallback
        playWithAudioPlayer(audioData);
      }
    }

    // Cleanup on unmount or when audioData changes
    return () => {
      const currentModel = modelRef.current;
      if (currentModel) {
        currentModel.stopSpeaking();
      }
      audioPlayerRef.current?.stop();
      if (lipSyncIntervalRef.current) {
        clearInterval(lipSyncIntervalRef.current);
        lipSyncIntervalRef.current = null;
      }
      if (currentAudioUrlRef.current) {
        URL.revokeObjectURL(currentAudioUrlRef.current);
        currentAudioUrlRef.current = null;
      }
    };
  }, [
    audioData,
    onAudioEnd,
    modelRef, // Try fallback
    playWithAudioPlayer,
  ]);

  // 初回レンダリング時にSoundManagerにパッチを適用
  useEffect(() => {
    // audioContextPatch.tsの便利関数を使用してSoundManagerを探してパッチを適用
    findAndPatchSoundManager(PixiLive2D, Live2DModel);
  }, []);

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
            app.stage.addChild(model as unknown as PIXI.DisplayObject);

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
                model?.motion(MOTION_TAP);
              }
              // iOS/PWA対策: タップ時にAudioContextを初期化
              if (window.AudioContext || window.webkitAudioContext) {
                const AudioContextClass = window.AudioContext || window.webkitAudioContext;
                if (!window.audioContext) {
                  window.audioContext = new AudioContextClass();
                }
                if (window.audioContext.state === "suspended") {
                  window.audioContext.resume();
                }
              }
            });

            // Store model reference
            const currentModelRef = modelRef;
            currentModelRef.current = model;

            // Extract model information
            const extractModelInfo = () => {
              const parameters: ModelParameter[] = [];
              const motions: ModelMotion[] = [];
              const expressions: ModelExpression[] = [];

              try {
                // Extract parameters
                if (model?.internalModel?.coreModel) {
                  const coreModel = model.internalModel.coreModel as Live2DCoreModel;
                  const paramCount = coreModel.getParameterCount?.() || 0;

                  for (let i = 0; i < paramCount; i++) {
                    // Get parameter ID - this should return the actual parameter name
                    let paramId = "";
                    try {
                      // Try different methods to get the parameter ID
                      if (coreModel._parameterIds?.[i]) {
                        paramId = coreModel._parameterIds[i] || `param_${i}`;
                      } else if (coreModel.getParameterId) {
                        paramId = coreModel.getParameterId(i);
                      } else if (coreModel._model?._parameterIds?.[i]) {
                        paramId = coreModel._model._parameterIds[i] || `param_${i}`;
                      } else {
                        paramId = `param_${i}`;
                      }
                    } catch (_e) {
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
                if (model?.internalModel?.motionManager) {
                  const motionManager = model.internalModel.motionManager;
                  const motionGroups = motionManager.motionGroups || {};

                  for (const [group, groupMotions] of Object.entries(motionGroups)) {
                    if (Array.isArray(groupMotions)) {
                      groupMotions.forEach((_motion, index) => {
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
                        groupDefs.forEach((_def, index) => {
                          // Avoid duplicates
                          const exists = motions.find(
                            (m) => m.group === group && m.index === index,
                          );
                          if (!exists) {
                            motions.push({
                              group,
                              index,
                              name: `${group}_${index}`,
                            });
                          }
                        });
                      }
                    }
                  }
                }

                // Extract expressions from expression manager
                const extendedModel = model?.internalModel as ExtendedInternalModel;
                if (extendedModel?.expressionManager) {
                  const expressionManager = extendedModel.expressionManager;
                  const expressionDefinitions =
                    expressionManager.definitions || expressionManager.expressions || {};

                  for (const expName of Object.keys(expressionDefinitions)) {
                    expressions.push({ name: expName });
                  }

                  // Also check expression list if available
                  if (expressionManager.expressionList) {
                    expressionManager.expressionList.forEach((exp) => {
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
              if (model.internalModel?.focusController) {
                model.internalModel.focusController.focus(0, 0, true); // instant=true で即座に適用
              }

              // face_forward モーションがあるか確認
              try {
                // モーションを試行
                model.motion(MOTION_FACE_FORWARD);
              } catch {
                // face_forward モーションがない場合、パラメータを直接設定
                if (model.internalModel?.coreModel) {
                  const coreModel = model.internalModel.coreModel as Live2DCoreModel;

                  // まずGroupから"Focus"グループのパラメータを取得を試みる
                  let focusParams: string[] = [];
                  try {
                    // model3.jsonの設定からGroupsを取得
                    const extendedModel = model.internalModel as ExtendedInternalModel;
                    if (extendedModel?.settings?.groups) {
                      const focusGroup = extendedModel.settings.groups.find(
                        (g: Live2DModelGroup) =>
                          g.Name === GROUP_NAME_FOCUS || g.name === GROUP_NAME_FOCUS,
                      );
                      if (focusGroup?.Ids) {
                        focusParams = focusGroup.Ids;
                      }
                    }
                  } catch (_e) {}

                  // Focusグループが見つからない場合はデフォルトのパラメータを使用
                  if (focusParams.length === 0) {
                    focusParams = DEFAULT_FOCUS_PARAMS;
                  }

                  // パラメータをリセット
                  for (const paramName of focusParams) {
                    try {
                      const paramIndex = coreModel.getParameterIndex?.(paramName);
                      if (paramIndex >= 0) {
                        // すべて0にリセット
                        coreModel.setParameterValueByIndex?.(paramIndex, 0);
                      }
                    } catch (_e) {
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
            model.motion(MOTION_IDLE);
          }
        } catch (_error) {
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
      const currentModelRef = modelRef;
      if (currentModelRef.current) {
        currentModelRef.current = null;
      }
      if (appRef.current) {
        appRef.current.destroy(true);
        appRef.current = null;
      }
    };
  }, [width, height, onModelLoaded, onModelInfoUpdate, modelRef]);

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
