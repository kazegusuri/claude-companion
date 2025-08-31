import { ActionIcon, Stack, Tooltip } from "@mantine/core";
import { IconMessage, IconMessageDown, IconMessageOff } from "@tabler/icons-react";
import type React from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import { ChatDisplay } from "../components/ChatDisplay";
import { MainLayout } from "../components/Layout/MainLayout";
import { Live2DModelViewer } from "../components/Live2DModelViewer";
import type { ChatMessage, ConnectionStatus } from "../services/WebSocketClient";
import { WebSocketAudioClient } from "../services/WebSocketClient";

type BubbleState = "right" | "bottom" | "hidden";

export const Dashboard: React.FC = () => {
  const [speechText, setSpeechText] = useState("音声を待機中...");
  const [bubbleState, setBubbleState] = useState<BubbleState>("bottom"); // 初期状態で下側表示
  const [isAudioEnabled, setIsAudioEnabled] = useState(false);
  const [_connectionStatus, setConnectionStatus] = useState<ConnectionStatus>("disconnected");
  const [currentMessageId, setCurrentMessageId] = useState<string | null>(null);
  const [currentAudioData, setCurrentAudioData] = useState<string | undefined>(undefined);

  const wsClient = useRef<WebSocketAudioClient | null>(null);
  const audioQueue = useRef<ChatMessage[]>([]);
  const isProcessingQueue = useRef(false);

  // 音声キューを処理
  const processAudioQueue = useCallback(async () => {
    if (isProcessingQueue.current || audioQueue.current.length === 0) {
      return;
    }

    if (!isAudioEnabled) {
      audioQueue.current = [];
      setCurrentAudioData(undefined);
      return;
    }

    isProcessingQueue.current = true;
    const message = audioQueue.current[0];

    if (message?.audioData) {
      setCurrentMessageId(message?.id || null);
      // Live2DModelViewerのspeakメソッドで再生
      setCurrentAudioData(message.audioData);
    } else {
      audioQueue.current.shift();
      isProcessingQueue.current = false;
    }
  }, [isAudioEnabled]);

  // WebSocketメッセージハンドラー
  const handleWebSocketMessage = useCallback(
    (message: ChatMessage) => {
      // テキストを更新（assistantメッセージのみ）
      if (message.text && message.role === "assistant") {
        setSpeechText(message.text);
      }

      // 音声データがある場合はキューに追加
      // Check for assistant messages with audio subtype or legacy audio type
      if (
        (message.type === "audio" ||
          (message.type === "assistant" && message.subType === "audio")) &&
        message.audioData &&
        isAudioEnabled
      ) {
        // 既存のメッセージがキューにないか確認
        if (!audioQueue.current.some((msg) => msg.id === message.id)) {
          audioQueue.current.push(message);
          // 優先度でソート
          audioQueue.current.sort((a, b) => b.priority - a.priority);
          // キューの処理を開始
          processAudioQueue();
        }
      }
    },
    [isAudioEnabled, processAudioQueue],
  );

  // 音声再生終了時の処理
  const handleAudioEnd = useCallback(() => {
    // キューから削除
    audioQueue.current.shift();
    setCurrentMessageId(null);
    setCurrentAudioData(undefined);
    isProcessingQueue.current = false;

    // 次のアイテムを処理
    if (audioQueue.current.length > 0) {
      setTimeout(processAudioQueue, 100);
    }
  }, [processAudioQueue]);

  // WebSocket接続の初期化
  useEffect(() => {
    // 既存の接続をクリーンアップ
    if (wsClient.current) {
      wsClient.current.disconnect();
      wsClient.current = null;
    }

    // WebSocketクライアントを作成
    const wsUrl = import.meta.env.VITE_WS_URL || "ws://localhost:8080/ws/audio";
    wsClient.current = new WebSocketAudioClient(wsUrl, handleWebSocketMessage, setConnectionStatus);

    // WebSocketに接続
    wsClient.current.connect();

    // クリーンアップ
    return () => {
      if (wsClient.current) {
        wsClient.current.disconnect();
        wsClient.current = null;
      }
    };
  }, [handleWebSocketMessage]);

  // 音声のトグル
  const handleToggleAudio = async () => {
    if (!isAudioEnabled) {
      setIsAudioEnabled(true);
    } else {
      setIsAudioEnabled(false);
      audioQueue.current = [];
      setCurrentMessageId(null);
      setCurrentAudioData(undefined);
    }
  };

  // 3段階トグル: 右側 → 下側 → 非表示 → 右側...
  const toggleBubble = () => {
    setBubbleState((prev) => {
      switch (prev) {
        case "right":
          return "bottom";
        case "bottom":
          return "hidden";
        case "hidden":
          return "right";
      }
    });
  };

  // アイコンとツールチップのテキストを決定
  const getIconAndTooltip = () => {
    switch (bubbleState) {
      case "right":
        return {
          icon: <IconMessage size={18} />,
          tooltip: "吹き出し：右側表示中 → クリックで下側へ",
        };
      case "bottom":
        return {
          icon: <IconMessageDown size={18} />,
          tooltip: "吹き出し：下側表示中 → クリックで非表示",
        };
      case "hidden":
        return {
          icon: <IconMessageOff size={18} />,
          tooltip: "吹き出し：非表示 → クリックで右側へ",
        };
    }
  };

  const { icon, tooltip } = getIconAndTooltip();

  return (
    <MainLayout
      modelComponent={
        <div
          style={{
            flex: 1,
            minHeight: 0,
            display: "flex",
            flexDirection: "row",
            justifyContent: "space-between",
            padding: "10px",
          }}
        >
          <div
            style={{
              flex: 1,
              height: "100%",
              minHeight: 0,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <Live2DModelViewer
              speechText={speechText}
              isSpeaking={bubbleState !== "hidden"}
              bubbleSide={bubbleState === "hidden" ? "bottom" : bubbleState}
              useCard={true}
              cardTitle="ASSISTANT"
              {...(currentAudioData !== undefined && { audioData: currentAudioData })}
              onAudioEnd={handleAudioEnd}
            />
          </div>
          <div
            style={{
              minHeight: 0,
              display: "flex",
              flexDirection: "column",
              alignItems: "flex-end",
              justifyContent: "flex-end",
              paddingBottom: "5px",
            }}
          >
            <Stack gap="xs">
              <Tooltip label={isAudioEnabled ? "音声ON" : "音声OFF"} position="top" withArrow>
                <ActionIcon
                  onClick={handleToggleAudio}
                  size="sm"
                  radius="xl"
                  variant={isAudioEnabled ? "filled" : "light"}
                  color={isAudioEnabled ? "green" : "gray"}
                  style={{
                    zIndex: 1001,
                  }}
                >
                  {isAudioEnabled ? "🔊" : "🔇"}
                </ActionIcon>
              </Tooltip>
              <Tooltip label={tooltip} position="top" withArrow>
                <ActionIcon
                  onClick={toggleBubble}
                  size="sm"
                  radius="xl"
                  variant="filled"
                  style={{
                    zIndex: 1001,
                  }}
                >
                  {icon}
                </ActionIcon>
              </Tooltip>
            </Stack>
          </div>
        </div>
      }
      scheduleComponent={null}
      textComponent={null}
      chatComponent={<ChatDisplay currentPlayingMessageId={currentMessageId} />}
    />
  );
};
