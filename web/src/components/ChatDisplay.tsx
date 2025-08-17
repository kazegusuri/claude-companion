import React, { useEffect, useRef, useState, useCallback } from "react";
import { Stack, Text, Paper, Badge, ScrollArea, Button, Group, Box } from "@mantine/core";
import { WebSocketAudioClient } from "../services/WebSocketClient";
import type { ConnectionStatus, AudioMessage } from "../services/WebSocketClient";
import { AudioPlayer } from "../services/AudioPlayer";

interface MessageHistory {
  id: string;
  text: string;
  timestamp: Date;
  metadata?: AudioMessage["metadata"];
  isPlaying?: boolean;
}

export const ChatDisplay: React.FC = () => {
  const [messages, setMessages] = useState<MessageHistory[]>([]);
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>("disconnected");
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentMessageId, setCurrentMessageId] = useState<string | null>(null);
  const [audioQueue, setAudioQueue] = useState<AudioMessage[]>([]);
  const [isAudioEnabled, setIsAudioEnabled] = useState(false);
  const [lastReceivedMessage, setLastReceivedMessage] = useState<AudioMessage | null>(null);

  const wsClient = useRef<WebSocketAudioClient | null>(null);
  const audioPlayer = useRef<AudioPlayer | null>(null);
  const viewportRef = useRef<HTMLDivElement>(null); // ScrollAreaのviewport参照
  const isProcessingQueue = useRef(false);

  // Configuration
  const MAX_MESSAGES = 100;

  // Initialize WebSocket connection
  useEffect(() => {
    if (wsClient.current) {
      wsClient.current.disconnect();
      wsClient.current = null;
    }
    if (audioPlayer.current) {
      audioPlayer.current.stop();
      audioPlayer.current = null;
    }

    const wsUrl = import.meta.env.VITE_WS_URL || "ws://localhost:8080/ws/audio";
    wsClient.current = new WebSocketAudioClient(
      wsUrl,
      (message: AudioMessage) => {
        console.log("Received WebSocket message:", message);

        // Add to message history with max limit
        setMessages((prev) => {
          if (prev.some((msg) => msg.id === message.id)) {
            return prev;
          }

          const historyItem: MessageHistory = {
            id: message.id,
            text: message.text,
            timestamp: new Date(message.timestamp),
            metadata: message.metadata,
          };

          // Add new message and limit to MAX_MESSAGES (keep only the latest 100)
          const newMessages = [...prev, historyItem];
          if (newMessages.length > MAX_MESSAGES) {
            // Remove oldest messages to maintain the limit
            return newMessages.slice(newMessages.length - MAX_MESSAGES);
          }
          return newMessages;
        });

        // Store the last received message for audio processing
        setLastReceivedMessage(message);

        // Auto-scroll to bottom using viewportRef
        setTimeout(() => {
          if (viewportRef.current) {
            // ScrollAreaのviewportを最下部までスクロール
            viewportRef.current.scrollTop = viewportRef.current.scrollHeight;
          }
        }, 100);
      },
      setConnectionStatus,
    );
    wsClient.current.connect();

    return () => {
      if (wsClient.current) {
        wsClient.current.disconnect();
        wsClient.current = null;
      }
      if (audioPlayer.current) {
        audioPlayer.current.stop();
        audioPlayer.current = null;
      }
    };
  }, []);

  // Handle new message for audio queue
  useEffect(() => {
    if (
      lastReceivedMessage &&
      isAudioEnabled &&
      lastReceivedMessage.type === "audio" &&
      lastReceivedMessage.audioData
    ) {
      console.log("Adding audio to queue from lastReceivedMessage");
      setAudioQueue((prev) => {
        if (prev.some((msg) => msg.id === lastReceivedMessage.id)) {
          return prev;
        }
        const newQueue = [...prev, lastReceivedMessage];
        newQueue.sort((a, b) => b.priority - a.priority);
        return newQueue;
      });
    }
  }, [lastReceivedMessage, isAudioEnabled]);

  // Process audio queue
  useEffect(() => {
    if (audioQueue.length > 0 && !isProcessingQueue.current && isAudioEnabled) {
      processNextInQueue();
    }
  }, [audioQueue, isAudioEnabled]);

  const processNextInQueue = async () => {
    if (isProcessingQueue.current || audioQueue.length === 0) {
      return;
    }

    // Skip audio playback if globally disabled
    if (!isAudioEnabled) {
      console.log("Audio playback disabled - skipping queue");
      setAudioQueue([]); // Clear the queue
      return;
    }

    isProcessingQueue.current = true;
    const message = audioQueue[0];

    if (message.audioData) {
      // Create audio player if not exists
      if (!audioPlayer.current) {
        audioPlayer.current = new AudioPlayer();
        audioPlayer.current.setVolume(0.8); // Set default volume
      }

      setCurrentMessageId(message.id);
      setIsPlaying(true);
      setMessages((prev) =>
        prev.map((msg) => (msg.id === message.id ? { ...msg, isPlaying: true } : msg)),
      );

      try {
        await audioPlayer.current.playBase64Audio(message.audioData, {
          onEnd: () => {
            setAudioQueue((prev) => prev.slice(1));
            setIsPlaying(false);
            setCurrentMessageId(null);
            setMessages((prev) =>
              prev.map((msg) => (msg.id === message.id ? { ...msg, isPlaying: false } : msg)),
            );
            isProcessingQueue.current = false;

            if (audioQueue.length > 1) {
              setTimeout(processNextInQueue, 100);
            }
          },
          onError: (error) => {
            console.warn("Audio playback skipped:", error.message);
            setAudioQueue((prev) => prev.slice(1));
            setMessages((prev) =>
              prev.map((msg) => (msg.id === message.id ? { ...msg, isPlaying: false } : msg)),
            );
            isProcessingQueue.current = false;
            setIsPlaying(false);
            setCurrentMessageId(null);

            if (audioQueue.length > 1) {
              setTimeout(processNextInQueue, 100);
            }
          },
        });
      } catch (error) {
        console.error("Failed to play audio:", error);
        setAudioQueue((prev) => prev.slice(1));
        isProcessingQueue.current = false;
        setIsPlaying(false);
        setCurrentMessageId(null);
      }
    } else {
      setAudioQueue((prev) => prev.slice(1));
      isProcessingQueue.current = false;
    }
  };

  const handleToggleAudio = async () => {
    if (!isAudioEnabled) {
      try {
        if (!audioPlayer.current) {
          audioPlayer.current = new AudioPlayer();
        }
        await audioPlayer.current.ensureInitialized();

        if (audioPlayer.current.isContextSuspended()) {
          console.warn("AudioContext is suspended - cannot enable audio");
          alert("音声を有効にできません。ページをリロードしてからもう一度お試しください。");
          return;
        }

        setIsAudioEnabled(true);
        console.log("Audio enabled successfully");
      } catch (error) {
        console.error("Failed to enable audio:", error);
        alert("音声の初期化に失敗しました。");
      }
    } else {
      setIsAudioEnabled(false);
      audioPlayer.current?.stop();
      setAudioQueue([]);
      setIsPlaying(false);
      setCurrentMessageId(null);
      console.log("Audio disabled");
    }
  };

  const handleStop = () => {
    audioPlayer.current?.stop();
    setAudioQueue([]);
    isProcessingQueue.current = false;
    setIsPlaying(false);
    setCurrentMessageId(null);
    setMessages((prev) => prev.map((msg) => ({ ...msg, isPlaying: false })));
  };

  const handleClearHistory = () => {
    setMessages([]);
  };

  const handleReconnect = () => {
    wsClient.current?.connect();
  };

  const formatTime = (date: Date) => {
    return date.toLocaleTimeString("ja-JP", {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  };

  const getEventTypeIcon = (eventType?: string) => {
    switch (eventType) {
      case "tool":
        return "🔧";
      case "message":
        return "💬";
      case "system":
        return "ℹ️";
      case "error":
        return "⚠️";
      default:
        return "📝";
    }
  };

  const getStatusColor = () => {
    switch (connectionStatus) {
      case "connected":
        return "green";
      case "connecting":
        return "yellow";
      case "error":
      case "failed":
        return "red";
      default:
        return "gray";
    }
  };

  // 追従の閾値
  const NEAR_BOTTOM_PX = 16;

  const handleScrollPosChange = ({ y }: { x: number; y: number }) => {
    const v = viewportRef.current;
    if (!v) return;
    const atBottom = v.scrollHeight - (y + v.clientHeight) <= NEAR_BOTTOM_PX;
    setFollow(atBottom);
  };

  return (
    <Box
      style={{
        display: "flex",
        flexDirection: "column",
        flex: 1,
        minHeight: 0,
        overflow: "hidden",
      }}
    >
      <Group
        gap="xs"
        justify="space-between"
        p="xs"
        style={{
          flex: "0 0 auto", // 固定高さ
          borderBottom: "1px solid rgba(255, 255, 255, 0.1)",
        }}
      >
        <Group gap="xs">
          <Badge color={getStatusColor()} variant="filled" size="sm">
            {connectionStatus === "connected"
              ? "接続中"
              : connectionStatus === "connecting"
                ? "接続中..."
                : connectionStatus === "disconnected"
                  ? "切断"
                  : connectionStatus === "error"
                    ? "エラー"
                    : connectionStatus === "failed"
                      ? "接続失敗"
                      : connectionStatus}
          </Badge>
          {audioQueue.length > 0 && (
            <Badge color="blue" variant="light" size="sm">
              キュー: {audioQueue.length}
            </Badge>
          )}
          {messages.length > 0 && (
            <Badge color="gray" variant="light" size="sm">
              メッセージ: {messages.length}/{MAX_MESSAGES}
            </Badge>
          )}
        </Group>
        <Group gap="xs">
          <Button
            size="xs"
            variant={isAudioEnabled ? "filled" : "light"}
            color={isAudioEnabled ? "green" : "gray"}
            onClick={handleToggleAudio}
          >
            {isAudioEnabled ? "🔊 音声ON" : "🔇 音声OFF"}
          </Button>
          {isPlaying && (
            <Button size="xs" color="red" onClick={handleStop}>
              ■ 停止
            </Button>
          )}
          <Button size="xs" variant="subtle" onClick={handleClearHistory}>
            クリア
          </Button>
          {(connectionStatus === "disconnected" || connectionStatus === "failed") && (
            <Button size="xs" variant="light" onClick={handleReconnect}>
              再接続
            </Button>
          )}
        </Group>
      </Group>

      {/* メッセージ一覧：中段だけをスクロールさせる */}
      <ScrollArea
        style={{
          flex: 1, // 残りの高さを全て使用
          minHeight: 0, // スクロール領域の高さ計算を正しく行う
        }}
        className="chat-scroll-container"
        data-testid="chat-scroll-area"
        viewportRef={viewportRef} // viewportRefで直接制御
        onScrollPositionChange={handleScrollPosChange}
        offsetScrollbars
        scrollbarSize={8}
      >
        <Stack gap="xs" p="sm">
          {messages.map((message) => (
            <Paper
              key={message.id}
              p="sm"
              style={{
                background:
                  message.id === currentMessageId
                    ? "rgba(255, 255, 255, 0.25)"
                    : "rgba(255, 255, 255, 0.2)",
                transition: "background 0.3s",
              }}
            >
              <Group gap="xs" justify="space-between" mb="xs">
                <Group gap="xs">
                  <Text size="xs" c="white" opacity={0.7}>
                    {formatTime(message.timestamp)}
                  </Text>
                  {message.metadata?.eventType && (
                    <Badge size="xs" variant="light">
                      {getEventTypeIcon(message.metadata.eventType)}
                      {message.metadata.eventType}
                    </Badge>
                  )}
                  {message.metadata?.toolName && (
                    <Badge size="xs" variant="light" color="blue">
                      {message.metadata.toolName}
                    </Badge>
                  )}
                </Group>
                {message.id === currentMessageId && (
                  <Badge size="xs" color="green" variant="dot">
                    再生中
                  </Badge>
                )}
              </Group>
              <Text size="sm" c="white" style={{ lineHeight: 1.5 }}>
                {message.text}
              </Text>
            </Paper>
          ))}
        </Stack>
      </ScrollArea>

      {/* 将来的に入力欄を追加する場合はここに */}
      {/* <Group p="xs" gap="xs" style={{ flex: "0 0 auto" }}>
        <TextInput style={{ flex: 1 }} placeholder="Type message..." />
        <Button>Send</Button>
      </Group> */}
    </Box>
  );
};
