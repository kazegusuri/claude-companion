import React, { useEffect, useRef, useState } from "react";
import { Stack, Text, Paper, Badge, ScrollArea, Button, Group, Box } from "@mantine/core";
import { WebSocketAudioClient } from "../services/WebSocketClient";
import type { ConnectionStatus, AudioMessage } from "../services/WebSocketClient";

interface MessageHistory {
  id: string;
  text: string;
  timestamp: Date;
  metadata?: AudioMessage["metadata"];
}

interface ChatDisplayProps {
  currentPlayingMessageId?: string | null;
  onMessagesUpdate?: (messages: AudioMessage[]) => void;
}

export const ChatDisplay: React.FC<ChatDisplayProps> = ({
  currentPlayingMessageId,
  onMessagesUpdate,
}) => {
  const [messages, setMessages] = useState<MessageHistory[]>([]);
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>("disconnected");

  const wsClient = useRef<WebSocketAudioClient | null>(null);
  const viewportRef = useRef<HTMLDivElement>(null); // ScrollAreaのviewport参照

  // Configuration
  const MAX_MESSAGES = 100;

  // Initialize WebSocket connection
  useEffect(() => {
    if (wsClient.current) {
      wsClient.current.disconnect();
      wsClient.current = null;
    }

    const wsUrl = import.meta.env["VITE_WS_URL"] || "ws://localhost:8080/ws/audio";
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

        // 親コンポーネントにメッセージを通知
        if (onMessagesUpdate) {
          onMessagesUpdate([message]);
        }

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
    };
  }, [onMessagesUpdate]);

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
        <Text size="md" fw={600} c="white">
          Chat
        </Text>
        <Group gap="xs">
          <Badge color={getStatusColor()} variant="filled" size="sm">
            {connectionStatus === "connected"
              ? "接続中"
              : connectionStatus === "connecting"
                ? "接続中..."
                : connectionStatus === "disconnected" || connectionStatus === "failed"
                  ? "切断"
                  : connectionStatus === "error"
                    ? "再接続待機中"
                    : connectionStatus}
          </Badge>
          {messages.length > 0 && (
            <Badge color="gray" variant="light" size="sm">
              メッセージ: {messages.length}/{MAX_MESSAGES}
            </Badge>
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
                  message.id === currentPlayingMessageId
                    ? "rgba(139, 92, 246, 0.3)" // 紫色のハイライト
                    : "rgba(255, 255, 255, 0.2)",
                border:
                  message.id === currentPlayingMessageId
                    ? "2px solid rgba(139, 92, 246, 0.8)"
                    : "none",
                transition: "all 0.3s",
                transform: message.id === currentPlayingMessageId ? "scale(1.02)" : "scale(1)",
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
                {message.id === currentPlayingMessageId && (
                  <Badge size="xs" color="violet" variant="filled">
                    🔊 再生中
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
