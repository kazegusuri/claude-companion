import React, { useState, useEffect, useRef, useCallback, useMemo } from "react";
import { Box } from "@mantine/core";
import { useSearchParams } from "react-router-dom";
import { Live2DModelViewer } from "../components/Live2DModelViewer";
import { ChatDisplay } from "../components/ChatDisplay";
import { WebSocketAudioClient } from "../services/WebSocketClient";
import type { ConnectionStatus, ChatMessage } from "../services/WebSocketClient";
import styles from "./MobileDashboard.module.css";

export const MobileDashboard: React.FC = () => {
  const [searchParams] = useSearchParams();
  const [speechText, setSpeechText] = useState("音声を待機中...");
  const [_connectionStatus, setConnectionStatus] = useState<ConnectionStatus>("disconnected");
  const [currentMessageId, setCurrentMessageId] = useState<string | null>(null);
  const [windowSize, setWindowSize] = useState({ width: 0, height: 0 });
  const [isAudioEnabled, setIsAudioEnabled] = useState(false);
  const [currentAudioData, setCurrentAudioData] = useState<string | undefined>(undefined);

  // URLパラメータから指定された幅と高さを取得（デフォルトは400x1280）
  const specifiedDimensions = useMemo(() => {
    const width = parseInt(searchParams.get("width") || "400", 10);
    const height = parseInt(searchParams.get("height") || "1280", 10);
    return { width, height };
  }, [searchParams]);

  const wsClient = useRef<WebSocketAudioClient | null>(null);
  const audioQueue = useRef<ChatMessage[]>([]);
  const isProcessingQueue = useRef(false);

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
  }, []);

  // 音声キューを処理（useCallbackでメモ化）
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

    if (message && message.audioData) {
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

      // 音声データがある場合はキューに追加（音声が有効な場合のみ）
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

  // 音声出力のトグル
  const handleAudioToggle = useCallback(() => {
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
  }, []);

  // WebSocket接続の初期化
  useEffect(() => {
    // 既存の接続をクリーンアップ
    if (wsClient.current) {
      wsClient.current.disconnect();
      wsClient.current = null;
    }

    // WebSocketクライアントを作成
    const wsUrl = import.meta.env["VITE_WS_URL"] || "ws://localhost:8080/ws/audio";
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
      className={styles["mobileContainer"] || ""}
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
        className={styles["live2dSection"] || ""}
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          padding: "4px",
          boxSizing: "border-box",
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
          {...(currentAudioData !== undefined && { audioData: currentAudioData })}
          onAudioEnd={handleAudioEnd}
        />
      </Box>

      {/* 下段: Chat Component (高さ512px = 40%) */}
      <Box
        className={styles["chatSection"] || ""}
        style={{
          borderTop: "1px solid rgba(255, 255, 255, 0.1)",
          boxSizing: "border-box",
        }}
      >
        <ChatDisplay
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
