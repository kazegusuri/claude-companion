import React, { useEffect, useRef, useState } from "react";
import {
  Stack,
  Text,
  Paper,
  Badge,
  ScrollArea,
  Button,
  Group,
  Box,
  Textarea,
  ActionIcon,
} from "@mantine/core";
import { IconSend, IconCheck, IconX, IconVolume, IconVolumeOff } from "@tabler/icons-react";
import { WebSocketAudioClient } from "../services/WebSocketClient";
import type { ConnectionStatus, ChatMessage } from "../services/WebSocketClient";

interface MessageHistory {
  id: string;
  text: string;
  timestamp: Date;
  metadata?: ChatMessage["metadata"];
  role?: "system" | "user" | "assistant";
  subType?: "audio" | "text";
}

interface ChatDisplayProps {
  currentPlayingMessageId?: string | null;
  onMessagesUpdate?: (messages: ChatMessage[]) => void;
  variant?: "default" | "mobile";
  maxDisplayMessages?: number;
  showInput?: boolean;
  onAudioToggle?: () => void;
  isAudioEnabled?: boolean;
}

export const ChatDisplay: React.FC<ChatDisplayProps> = ({
  currentPlayingMessageId,
  onMessagesUpdate,
  variant = "default",
  maxDisplayMessages,
  showInput = true,
  onAudioToggle,
  isAudioEnabled = false,
}) => {
  const [messages, setMessages] = useState<MessageHistory[]>([]);
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>("disconnected");
  const [inputMessage, setInputMessage] = useState("");
  const [respondedPermissions, setRespondedPermissions] = useState<Set<string>>(new Set());
  const [currentSessionId, setCurrentSessionId] = useState<string | null>(null);

  const wsClient = useRef<WebSocketAudioClient | null>(null);
  const viewportRef = useRef<HTMLDivElement>(null); // ScrollAreaのviewport参照

  // Configuration
  const MAX_MESSAGES = variant === "mobile" ? 50 : 100;

  // Initialize WebSocket connection
  useEffect(() => {
    if (wsClient.current) {
      wsClient.current.disconnect();
      wsClient.current = null;
    }

    const wsUrl = import.meta.env["VITE_WS_URL"] || "ws://localhost:8080/ws/audio";
    wsClient.current = new WebSocketAudioClient(
      wsUrl,
      (message: ChatMessage) => {
        // Track sessionId from any message that has it
        if (message.metadata?.sessionId) {
          setCurrentSessionId(message.metadata.sessionId);
        }

        // Add to message history with max limit
        setMessages((prev) => {
          if (prev.some((msg) => msg.id === message.id)) {
            return prev;
          }

          // Format text based on event type
          let displayText = message.text;
          if (message.metadata?.eventType === "tool_permission") {
            displayText = `🔐 ${message.text}`;
          } else if (message.metadata?.eventType === "command_success") {
            displayText = `✅ ${message.text}`;
          } else if (message.metadata?.eventType === "command_error") {
            displayText = `❌ ${message.text}`;
          }

          const historyItem: MessageHistory = {
            id: message.id,
            text: displayText,
            timestamp: new Date(message.timestamp),
            metadata: message.metadata,
            ...(message.role && { role: message.role }),
            ...(message.subType && { subType: message.subType }),
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

  const handleSendMessage = () => {
    if (!inputMessage.trim() || !wsClient.current || !currentSessionId) {
      if (!currentSessionId) {
        console.error("Session ID not available yet. Waiting for tool_permission message.");
      }
      return;
    }

    // WebSocket経由でメッセージを送信（currentSessionIdを使用）
    wsClient.current.sendMessage(inputMessage.trim(), currentSessionId);

    // 入力フィールドをクリア
    setInputMessage("");
  };

  const handleKeyPress = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // Shift+Enterで改行、Enterのみで送信
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      handleSendMessage();
    }
  };

  const handlePermissionResponse = (
    messageId: string,
    action: "permit" | "deny",
    sessionId?: string,
  ) => {
    if (!wsClient.current) return;

    // Use sessionId from the message metadata if available, otherwise use currentSessionId
    const targetSessionId = sessionId || currentSessionId;
    if (!targetSessionId) {
      console.error("Session ID not available for permission response");
      return;
    }

    // Send confirmation response with the appropriate sessionId
    wsClient.current.sendConfirmResponse(action, messageId, targetSessionId);

    // Mark this permission as responded
    setRespondedPermissions((prev) => new Set(prev).add(messageId));
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
      case "tool_permission":
        return "🔐";
      case "command_success":
        return "✅";
      case "command_error":
        return "❌";
      case "user_message_echo":
        return "💭";
      default:
        return "📝";
    }
  };

  const getRoleIcon = (role?: string) => {
    switch (role) {
      case "user":
        return "👤";
      case "assistant":
        return "🤖";
      case "system":
        return "⚙️";
      default:
        return "💬";
    }
  };

  const getRoleColor = (role?: string) => {
    switch (role) {
      case "user":
        return "blue";
      case "assistant":
        return "green";
      case "system":
        return "gray";
      default:
        return "gray";
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
        p={variant === "mobile" ? "4px 8px" : "xs"}
        style={{
          flex: "0 0 auto", // 固定高さ
          borderBottom: "1px solid rgba(255, 255, 255, 0.1)",
        }}
      >
        <Text size={variant === "mobile" ? "sm" : "md"} fw={600} c="white">
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
          {variant === "mobile" && onAudioToggle && (
            <ActionIcon
              size="sm"
              variant="subtle"
              color={isAudioEnabled ? "green" : "gray"}
              onClick={onAudioToggle}
              title={isAudioEnabled ? "音声出力ON" : "音声出力OFF"}
            >
              {isAudioEnabled ? <IconVolume size={16} /> : <IconVolumeOff size={16} />}
            </ActionIcon>
          )}
          {variant !== "mobile" && messages.length > 0 && (
            <Badge color="gray" variant="light" size="sm">
              メッセージ: {messages.length}/{MAX_MESSAGES}
            </Badge>
          )}
          {variant !== "mobile" && (
            <Button size="xs" variant="subtle" onClick={handleClearHistory}>
              クリア
            </Button>
          )}
          {variant !== "mobile" &&
            (connectionStatus === "disconnected" || connectionStatus === "failed") && (
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
        <Stack gap={variant === "mobile" ? "4px" : "xs"} p={variant === "mobile" ? "4px" : "sm"}>
          {/* モバイルの場合は最新N件のみ表示 */}
          {(maxDisplayMessages ? messages.slice(-maxDisplayMessages) : messages).map((message) => (
            <Paper
              key={message.id}
              p={variant === "mobile" ? "8px" : "sm"}
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
              <Group
                gap={variant === "mobile" ? "4px" : "xs"}
                justify="space-between"
                mb={variant === "mobile" ? "4px" : "xs"}
              >
                <Group gap={variant === "mobile" ? "4px" : "xs"}>
                  <Text
                    size="xs"
                    c="white"
                    opacity={0.7}
                    style={{ fontSize: variant === "mobile" ? "10px" : undefined }}
                  >
                    {formatTime(message.timestamp)}
                  </Text>
                  {message.role && (
                    <Badge
                      size="xs"
                      variant="filled"
                      color={getRoleColor(message.role)}
                      style={{
                        fontSize: variant === "mobile" ? "10px" : undefined,
                        padding: variant === "mobile" ? "2px 4px" : undefined,
                      }}
                    >
                      {getRoleIcon(message.role)} {message.role}
                    </Badge>
                  )}
                  {message.subType && (
                    <Badge
                      size="xs"
                      variant="light"
                      color="violet"
                      style={{
                        fontSize: variant === "mobile" ? "10px" : undefined,
                        padding: variant === "mobile" ? "2px 4px" : undefined,
                      }}
                    >
                      {message.subType === "audio" ? "🎤" : "📝"} {message.subType}
                    </Badge>
                  )}
                  {message.metadata?.eventType && !message.role && (
                    <Badge
                      size="xs"
                      variant="light"
                      color={
                        message.metadata.eventType === "tool_permission"
                          ? "yellow"
                          : message.metadata.eventType === "command_success"
                            ? "green"
                            : message.metadata.eventType === "command_error"
                              ? "red"
                              : "gray"
                      }
                    >
                      {getEventTypeIcon(message.metadata.eventType)}
                      {message.metadata.eventType}
                    </Badge>
                  )}
                  {message.metadata?.toolName && (
                    <Badge size="xs" variant="light" color="blue">
                      🔧 {message.metadata.toolName}
                    </Badge>
                  )}
                </Group>
                {message.id === currentPlayingMessageId && (
                  <Badge
                    size="xs"
                    color="violet"
                    variant="filled"
                    style={{
                      fontSize: variant === "mobile" ? "10px" : undefined,
                      padding: variant === "mobile" ? "2px 4px" : undefined,
                    }}
                  >
                    🔊 再生中
                  </Badge>
                )}
              </Group>
              <Text
                size={variant === "mobile" ? "xs" : "sm"}
                c="white"
                style={{
                  lineHeight: variant === "mobile" ? 1.3 : 1.5,
                  fontSize: variant === "mobile" ? "11px" : undefined,
                  marginTop: variant === "mobile" ? "2px" : undefined,
                }}
              >
                {message.text}
              </Text>

              {/* Permission buttons for tool_permission events */}
              {message.metadata?.eventType === "tool_permission" &&
                !respondedPermissions.has(message.id) && (
                  <Group gap="xs" mt="sm">
                    <Button
                      size="xs"
                      color="green"
                      leftSection={<IconCheck size={16} />}
                      onClick={() =>
                        handlePermissionResponse(message.id, "permit", message.metadata?.sessionId)
                      }
                    >
                      許可
                    </Button>
                    <Button
                      size="xs"
                      color="red"
                      leftSection={<IconX size={16} />}
                      onClick={() =>
                        handlePermissionResponse(message.id, "deny", message.metadata?.sessionId)
                      }
                    >
                      拒否
                    </Button>
                  </Group>
                )}
            </Paper>
          ))}
        </Stack>
      </ScrollArea>

      {/* メッセージ入力エリア（showInputがtrueの場合のみ表示） */}
      {showInput && (
        <Group
          p="xs"
          gap="xs"
          style={{
            flex: "0 0 auto",
            borderTop: "1px solid rgba(255, 255, 255, 0.1)",
          }}
        >
          <Textarea
            style={{ flex: 1 }}
            placeholder="メッセージを入力... (Enterで送信、Shift+Enterで改行)"
            value={inputMessage}
            onChange={(e) => setInputMessage(e.currentTarget.value)}
            onKeyDown={handleKeyPress}
            minRows={3}
            maxRows={6}
            autosize
            disabled={connectionStatus !== "connected" || !currentSessionId}
          />
          <ActionIcon
            size="lg"
            variant="filled"
            color="blue"
            onClick={handleSendMessage}
            disabled={!inputMessage.trim() || connectionStatus !== "connected" || !currentSessionId}
          >
            <IconSend size={18} />
          </ActionIcon>
        </Group>
      )}
    </Box>
  );
};
