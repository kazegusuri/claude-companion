import { Box } from "@mantine/core";
import type React from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { ChatDisplay } from "../components/ChatDisplay";
import { Live2DModelViewer } from "../components/Live2DModelViewer";
import type { ChatMessage, ConnectionStatus } from "../services/WebSocketClient";
import { WebSocketAudioClient } from "../services/WebSocketClient";
import styles from "./MobileDashboard.module.css";

export const MobileDashboard: React.FC = () => {
  const [searchParams] = useSearchParams();
  const [speechText, setSpeechText] = useState("音声を待機中...");
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>("disconnected");
  const [currentMessageId, setCurrentMessageId] = useState<string | null>(null);
  const [windowSize, setWindowSize] = useState({ width: 0, height: 0 });
  const [isAudioEnabled, setIsAudioEnabled] = useState(false); // 初期状態では無効（ユーザーインタラクションが必要）
  const [currentAudioData, setCurrentAudioData] = useState<string | undefined>(undefined);
  const [audioInitialized, setAudioInitialized] = useState(false); // AudioContext初期化状態

  // URLパラメータから指定された幅と高さを取得（デフォルトは400x1280）
  const specifiedDimensions = useMemo(() => {
    const width = parseInt(searchParams.get("width") || "400", 10);
    const height = parseInt(searchParams.get("height") || "1280", 10);
    return { width, height };
  }, [searchParams]);

  const wsClient = useRef<WebSocketAudioClient | null>(null);
  const audioQueue = useRef<ChatMessage[]>([]);
  const isProcessingQueue = useRef(false);

  // processAudioQueueのrefを作成
  const processAudioQueueRef = useRef<(() => void) | null>(null);

  // 音声が無効になったらキューをクリア
  useEffect(() => {
    if (!isAudioEnabled) {
      audioQueue.current = [];
      setCurrentMessageId(null);
      setCurrentAudioData(undefined);
    }
  }, [isAudioEnabled]);

  // isAudioEnabledをrefで管理して、関数の再作成を防ぐ
  const isAudioEnabledRef = useRef(isAudioEnabled);
  useEffect(() => {
    isAudioEnabledRef.current = isAudioEnabled;
  }, [isAudioEnabled]);

  // 音声キューを処理（一度だけ作成される関数）
  const processAudioQueue = useCallback(() => {
    if (isProcessingQueue.current || audioQueue.current.length === 0) {
      return;
    }

    if (!isAudioEnabledRef.current) {
      audioQueue.current = [];
      setCurrentAudioData(undefined);
      return;
    }

    const message = audioQueue.current[0];

    if (message?.audioData) {
      // メッセージの情報を保持してから削除
      const messageId = message.id;
      const audioData = message.audioData;

      // キューから削除
      audioQueue.current.shift();
      isProcessingQueue.current = true;

      // メッセージIDと音声データを設定
      setCurrentMessageId(messageId || null);
      setCurrentAudioData(audioData);
    } else {
      audioQueue.current.shift();
      // 次のメッセージを処理
      if (audioQueue.current.length > 0) {
        setTimeout(() => processAudioQueueRef.current?.(), 0);
      }
    }
  }, []); // 空の依存配列で一度だけ作成

  // processAudioQueueをrefに保存
  useEffect(() => {
    processAudioQueueRef.current = processAudioQueue;
  }, [processAudioQueue]);

  // 音声再生終了時の処理
  const handleAudioEnd = useCallback(() => {
    // 既にキューから削除済みなので、削除処理は不要
    setCurrentMessageId(null);
    setCurrentAudioData(undefined);
    isProcessingQueue.current = false;

    // 次のアイテムを処理
    if (audioQueue.current.length > 0) {
      setTimeout(() => processAudioQueueRef.current?.(), 100);
    }
  }, []);

  // WebSocketメッセージハンドラー（依存配列を空にして再作成を防ぐ）
  const handleWebSocketMessage = useCallback((message: ChatMessage) => {
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
      isAudioEnabledRef.current // refを使用
    ) {
      // 既存のメッセージがキューにないか確認
      const isDuplicate = audioQueue.current.some((msg) => msg.id === message.id);

      if (!isDuplicate) {
        audioQueue.current.push(message);

        // 優先度でソート（現在は全て同じ優先度なので実質的に追加順）
        audioQueue.current.sort((a, b) => b.priority - a.priority);

        // 処理中でない場合のみprocessAudioQueueを呼ぶ
        if (!isProcessingQueue.current) {
          processAudioQueueRef.current?.();
        }
      }
    }
  }, []); // 空の依存配列で一度だけ作成

  // 音声初期化（ユーザーインタラクションが必要）
  const initializeAudio = useCallback(() => {
    if (!audioInitialized) {
      // AudioContextを初期化（Live2DModelViewer内で処理される）
      setAudioInitialized(true);
      setIsAudioEnabled(true);
    }
  }, [audioInitialized]);

  // 音声出力のトグル
  const handleAudioToggle = useCallback(() => {
    // 初期化されていない場合は初期化を実行
    if (!audioInitialized) {
      initializeAudio();
      return;
    }

    setIsAudioEnabled((prev) => {
      const newState = !prev;

      // 音声を無効にした場合、再生中の音声を停止し、キューをクリア
      if (!newState) {
        audioQueue.current = [];
        setCurrentMessageId(null);
        setCurrentAudioData(undefined);
        isProcessingQueue.current = false;
      }

      return newState;
    });
  }, [audioInitialized, initializeAudio]);

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
      wsClient.current = client;
    }, 0);

    // クリーンアップ
    return () => {
      mounted = false;
      clearTimeout(timer);
      if (client) {
        client.disconnect();
        wsClient.current = null;
      }
    };
  }, []); // 空の依存配列で一度だけ実行

  // ビューポートスケーリング計算とサイズ追跡
  useEffect(() => {
    const updateScale = () => {
      const viewportWidth = window.innerWidth;
      const viewportHeight = window.innerHeight;

      // サイズ情報を更新
      setWindowSize({ width: viewportWidth, height: viewportHeight });

      if (viewportWidth < 400) {
        const scale = viewportWidth / 400;
        document.documentElement.style.setProperty("--scale-factor", scale.toString());
      } else {
        document.documentElement.style.setProperty("--scale-factor", "1");
      }
    };

    updateScale();
    window.addEventListener("resize", updateScale);
    return () => window.removeEventListener("resize", updateScale);
  }, []);

  return (
    <Box
      className={styles.mobileContainer || ""}
      style={{
        backgroundColor: "#1a1b1e",
        display: "flex",
        flexDirection: "column",
        position: "relative",
      }}
    >
      {/* デバッグ情報: 画面サイズ表示 */}
      <Box
        style={{
          position: "absolute",
          top: 0,
          left: 0,
          right: 0,
          backgroundColor: "rgba(0, 0, 0, 0.7)",
          color: "#00ff00",
          padding: "4px 8px",
          fontSize: "12px",
          fontFamily: "monospace",
          zIndex: 9999,
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
          gap: "16px",
        }}
      >
        <span style={{ color: searchParams.get("width") ? "#ffff00" : "#00ff00" }}>
          W: {searchParams.get("width") || windowSize.width}px
        </span>
        <span style={{ color: searchParams.get("height") ? "#ffff00" : "#00ff00" }}>
          H: {searchParams.get("height") || windowSize.height}px
        </span>
        <span>Ratio: {(windowSize.height / windowSize.width).toFixed(2)}</span>
        <span>{window.location.pathname === "/mobile" ? "Mobile Mode" : "Desktop Mode"}</span>
      </Box>
      {/* 上段: Live2D Model (高さ768px = 60%) */}
      <Box
        className={styles.live2dSection || ""}
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          padding: "4px",
          boxSizing: "border-box",
          position: "relative",
        }}
      >
        <Live2DModelViewer
          width={380}
          height={700}
          speechText={speechText}
          isSpeaking={true}
          bubbleSide="bottom"
          useCard={false}
          bubbleMaxWidth={360}
          specifiedWidth={specifiedDimensions.width}
          {...(currentAudioData
            ? { audioData: currentAudioData, audioMessageId: currentMessageId }
            : {})}
          onAudioEnd={handleAudioEnd}
        />
        {/* 音声初期化ボタン（未初期化時のみ表示） */}
        {!audioInitialized && (
          <button
            onClick={initializeAudio}
            style={{
              position: "absolute",
              bottom: "20px",
              left: "50%",
              transform: "translateX(-50%)",
              padding: "12px 24px",
              fontSize: "16px",
              fontWeight: "bold",
              backgroundColor: "#4CAF50",
              color: "white",
              border: "none",
              borderRadius: "8px",
              cursor: "pointer",
              boxShadow: "0 4px 6px rgba(0, 0, 0, 0.3)",
              zIndex: 100,
            }}
            type="button"
          >
            🔊 音声を有効にする
          </button>
        )}
      </Box>

      {/* 下段: Chat Component (高さ512px = 40%) */}
      <Box
        className={styles.chatSection || ""}
        style={{
          borderTop: "1px solid rgba(255, 255, 255, 0.1)",
          boxSizing: "border-box",
        }}
      >
        <ChatDisplay
          wsClient={wsClient.current}
          connectionStatus={connectionStatus}
          currentPlayingMessageId={currentMessageId}
          variant="mobile"
          maxDisplayMessages={3}
          showInput={false}
          onAudioToggle={handleAudioToggle}
          isAudioEnabled={isAudioEnabled}
        />
      </Box>
    </Box>
  );
};
