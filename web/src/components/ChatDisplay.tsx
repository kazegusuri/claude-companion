import {
  ActionIcon,
  Badge,
  Box,
  Button,
  Group,
  Paper,
  ScrollArea,
  Stack,
  Text,
  Textarea,
} from "@mantine/core";
import { IconCheck, IconSend, IconVolume, IconVolumeOff, IconX } from "@tabler/icons-react";
import type React from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import { messageRouter } from "../services/MessageRouter";
import type {
  ChatMessage,
  ConnectionStatus,
  WebSocketAudioClient,
} from "../services/WebSocketClient";

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
  agentPID?: number | null; // Agent PID for agent mode
  onAgentDisconnect?: () => void; // Callback when agent disconnects
  wsClient?: WebSocketAudioClient | null; // Shared WebSocket client instance
  connectionStatus?: ConnectionStatus; // Connection status from parent
}

export const ChatDisplay: React.FC<ChatDisplayProps> = ({
  currentPlayingMessageId,
  onMessagesUpdate: _onMessagesUpdate,
  variant = "default",
  maxDisplayMessages,
  showInput = true,
  onAudioToggle,
  isAudioEnabled = false,
  agentPID = null,
  onAgentDisconnect: _onAgentDisconnect,
  wsClient: externalWsClient,
  connectionStatus: externalConnectionStatus = "disconnected",
}) => {
  const [messages, setMessages] = useState<MessageHistory[]>([]);
  const [inputMessage, setInputMessage] = useState("");
  const [respondedPermissions, setRespondedPermissions] = useState<Set<string>>(new Set());
  const [currentSessionId, setCurrentSessionId] = useState<string | null>(null);

  const wsClient = externalWsClient; // Always use external client
  const connectionStatus = externalConnectionStatus; // Use external connection status
  const viewportRef = useRef<HTMLDivElement>(null); // ScrollAreaのviewport参照

  // Configuration
  const MAX_MESSAGES = variant === "mobile" ? 50 : 100;

  // WebSocketメッセージハンドラー
  const handleWebSocketMessage = useCallback(
    (message: ChatMessage) => {
      // メッセージルーターでフィルタリング
      if (!messageRouter.shouldAcceptMessage(message)) {
        return;
      }

      // Extract session ID from tool_permission messages
      if (message.metadata?.eventType === "tool_permission" && message.metadata?.sessionId) {
        setCurrentSessionId(message.metadata.sessionId);
      }

      // Convert ChatMessage to MessageHistory
      const newMessage: MessageHistory = {
        id: message.id,
        text: message.text,
        timestamp: new Date(message.timestamp),
        ...(message.metadata && { metadata: message.metadata }),
        ...(message.role && { role: message.role }),
        ...(message.subType && { subType: message.subType }),
      };

      // Add message to history
      setMessages((prev) => {
        const updated = [...prev, newMessage];
        // Keep only the last MAX_MESSAGES
        if (updated.length > MAX_MESSAGES) {
          return updated.slice(-MAX_MESSAGES);
        }
        return updated;
      });
    },
    [MAX_MESSAGES],
  );

  // WebSocketメッセージリスナーの登録
  useEffect(() => {
    if (!wsClient) return;

    const cleanup = wsClient.addMessageListener(handleWebSocketMessage);
    return cleanup;
  }, [wsClient, handleWebSocketMessage]);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    if (viewportRef.current) {
      viewportRef.current.scrollTop = viewportRef.current.scrollHeight;
    }
  }, [messages]);

  // Set or clear agent mode when agentPID changes
  useEffect(() => {
    if (!wsClient) return;

    if (agentPID) {
      wsClient.setAgentMode(agentPID);
      // Clear messages when switching to agent mode
      setMessages([]);
      setCurrentSessionId(null);
      setRespondedPermissions(new Set());
    } else {
      wsClient.clearAgentMode();
      // Clear messages when switching to timeline mode
      setMessages([]);
      setCurrentSessionId(null);
      setRespondedPermissions(new Set());
    }
  }, [agentPID, wsClient]);

  const handleClearHistory = () => {
    setMessages([]);
  };

  const handleReconnect = () => {
    wsClient?.connect();
  };

  const handleSendMessage = () => {
    if (!inputMessage.trim() || !wsClient || !currentSessionId) {
      return;
    }

    // WebSocket経由でメッセージを送信（currentSessionIdを使用）
    wsClient.sendMessage(inputMessage.trim(), currentSessionId);

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
    if (!wsClient) return;

    // Use sessionId from the message metadata if available, otherwise use currentSessionId
    const targetSessionId = sessionId || currentSessionId;
    if (!targetSessionId) {
      return;
    }

    // Send confirmation response with the appropriate sessionId
    wsClient.sendConfirmResponse(action, messageId, targetSessionId);

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
        <Group gap="xs">
          <Text size={variant === "mobile" ? "sm" : "md"} fw={600} c="white">
            Chat
          </Text>
          {agentPID && (
            <Badge color="blue" variant="light" size="sm">
              Agent PID: {agentPID}
            </Badge>
          )}
        </Group>
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
