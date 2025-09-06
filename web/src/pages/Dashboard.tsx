import type React from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import { AgentList } from "../components/AgentList";
import { ChatDisplay } from "../components/ChatDisplay";
import { MainLayout } from "../components/Layout/MainLayout";
import { Live2DModelViewer } from "../components/Live2DModelViewer";
import type { Agent } from "../services/AgentService";
import { messageRouter } from "../services/MessageRouter";
import type { ChatMessage, ConnectionStatus } from "../services/WebSocketClient";
import { WebSocketAudioClient } from "../services/WebSocketClient";

interface DashboardProps {
  isAudioEnabled: boolean;
}

export const Dashboard: React.FC<DashboardProps> = ({ isAudioEnabled }) => {
  const [speechText, setSpeechText] = useState("音声を待機中...");
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>("disconnected");
  const [currentMessageId, setCurrentMessageId] = useState<string | null>(null);
  const [currentAudioData, setCurrentAudioData] = useState<string | undefined>(undefined);
  const [selectedAgent, setSelectedAgent] = useState<Agent | null>(null); // Track selected agent
  // オーバーレイの位置管理
  const [overlayPosition, setOverlayPosition] = useState({ x: 250, y: window.innerHeight - 300 }); // オーバーレイの位置（初期は左下）
  const [isDragging, setIsDragging] = useState(false); // ドラッグ中かどうか
  const [dragOffset, setDragOffset] = useState({ x: 0, y: 0 }); // ドラッグのオフセット
  const [wsClient, setWsClient] = useState<WebSocketAudioClient | null>(null);

  const audioQueue = useRef<ChatMessage[]>([]);
  const isProcessingQueue = useRef(false);
  const processAudioQueueRef = useRef<(() => void) | null>(null);

  // 音声が無効になったらキューをクリア
  useEffect(() => {
    if (!isAudioEnabled) {
      audioQueue.current = [];
      setCurrentMessageId(null);
      setCurrentAudioData(undefined);
    }
  }, [isAudioEnabled]);

  // 音声キューを処理
  const processAudioQueue = useCallback(() => {
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

  // processAudioQueueをrefに保存
  useEffect(() => {
    processAudioQueueRef.current = processAudioQueue;
  }, [processAudioQueue]);

  // WebSocketメッセージハンドラー
  const handleWebSocketMessage = useCallback(
    (message: ChatMessage) => {
      // メッセージルーターでフィルタリング
      if (!messageRouter.shouldAcceptMessage(message)) {
        return;
      }

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
          processAudioQueueRef.current?.();
        }
      }
    },
    [isAudioEnabled],
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
      setTimeout(() => processAudioQueueRef.current?.(), 100);
    }
  }, []);

  // メッセージハンドラーのrefを作成
  const messageHandlerRef = useRef<(message: ChatMessage) => void>();

  // メッセージハンドラーをrefに保存
  useEffect(() => {
    messageHandlerRef.current = handleWebSocketMessage;
  }, [handleWebSocketMessage]);

  // WebSocket接続の初期化（単一接続を維持）
  useEffect(() => {
    let mounted = true;
    let client: WebSocketAudioClient | null = null;

    // 少し遅延させてStrictModeの二重レンダリングの影響を軽減
    const timer = setTimeout(() => {
      if (!mounted) return;

      // WebSocketクライアントを作成
      const wsUrl = import.meta.env.VITE_WS_URL || "ws://localhost:8080/ws/audio";
      client = new WebSocketAudioClient(
        wsUrl,
        (message) => messageHandlerRef.current?.(message),
        setConnectionStatus,
      );

      // WebSocketに接続
      client.connect();

      // stateにセット
      setWsClient(client);
    }, 0);

    // クリーンアップ
    return () => {
      mounted = false;
      clearTimeout(timer);
      if (client) {
        client.disconnect();
        setWsClient(null);
      }
    };
  }, []); // 空の依存配列で一度だけ実行

  // オーバーレイの位置に基づいて吹き出しの位置を決定（ドラッグ中も追従）
  const [overlayBubbleSide, setOverlayBubbleSide] = useState<"top" | "bottom" | "left" | "right">(
    "top",
  );

  useEffect(() => {
    const centerY = window.innerHeight / 2;

    // 画面を上下で分割して、オーバーレイがどの位置にあるか判定
    if (overlayPosition.y < centerY) {
      // 上半分にある場合は下に吹き出しを表示
      setOverlayBubbleSide("bottom");
    } else {
      // 下半分にある場合は上に吹き出しを表示
      setOverlayBubbleSide("top");
    }
  }, [overlayPosition]);

  // ドラッグ開始処理
  const handleMouseDown = (e: React.MouseEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsDragging(true);
    // 現在のオーバーレイ位置とマウス位置の差分を保存
    setDragOffset({
      x: e.clientX - overlayPosition.x,
      y: e.clientY - overlayPosition.y,
    });
  };

  // マウス移動処理（ドラッグ中）
  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (isDragging) {
        const newX = e.clientX - dragOffset.x;
        const newY = e.clientY - dragOffset.y;

        setOverlayPosition({
          x: newX,
          y: newY,
        });
      }
    };

    const handleMouseUp = () => {
      setIsDragging(false);
    };

    if (isDragging) {
      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);

      return () => {
        document.removeEventListener("mousemove", handleMouseMove);
        document.removeEventListener("mouseup", handleMouseUp);
      };
    }
    return undefined;
  }, [isDragging, dragOffset]);

  return (
    <>
      {/* オーバーレイLive2D（常に前面表示） */}
      <div
        style={{
          position: "fixed",
          left: `${overlayPosition.x}px`,
          top: `${overlayPosition.y}px`,
          transform: "translate(-50%, -50%)",
          zIndex: 9999,
          width: "500px",
          height: "600px",
          pointerEvents: "none", // クリックイベントを透過
          cursor: isDragging ? "grabbing" : "grab",
        }}
      >
        <div style={{ position: "relative", width: "100%", height: "100%" }}>
          {/* ドラッグ用の透明なハンドル領域（上部） */}
          {/* biome-ignore lint/a11y/useSemanticElements: ドラッグハンドル用の透明領域 */}
          <div
            role="button"
            tabIndex={-1}
            onMouseDown={handleMouseDown}
            style={{
              position: "absolute",
              top: 0,
              left: 0,
              right: 0,
              height: "120px", // ドラッグ可能エリアの高さ
              pointerEvents: "auto",
              cursor: isDragging ? "grabbing" : "grab",
              zIndex: 9998,
              backgroundColor: "transparent", // 完全に透明
            }}
          />
          <Live2DModelViewer
            width={500}
            height={600}
            speechText={speechText}
            isSpeaking={true}
            bubbleSide={overlayBubbleSide}
            useCard={true}
            cardTitle="ASSISTANT"
            {...(currentAudioData ? { audioData: currentAudioData } : {})}
            onAudioEnd={handleAudioEnd}
          />
        </div>
      </div>

      <MainLayout
        modelComponent={
          <AgentList
            onAgentClick={(agent) => {
              // Toggle agent selection
              if (selectedAgent?.pid === agent.pid) {
                // Clear agent mode
                setSelectedAgent(null);
                messageRouter.clearMode();
                wsClient?.clearAgentMode();
              } else {
                // Set agent mode
                setSelectedAgent(agent);
                messageRouter.setAgentMode(agent);
                wsClient?.setAgentMode(agent.pid);
              }
            }}
            selectedAgentPID={selectedAgent?.pid ?? null}
          />
        }
        scheduleComponent={null}
        textComponent={null}
        chatComponent={
          <ChatDisplay
            currentPlayingMessageId={currentMessageId}
            agentPID={selectedAgent?.pid ?? null}
            onAgentDisconnect={() => setSelectedAgent(null)}
            wsClient={wsClient}
            connectionStatus={connectionStatus}
          />
        }
      />
    </>
  );
};
