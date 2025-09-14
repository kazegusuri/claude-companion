import { useCallback, useEffect } from "react";
import type { ChatMessage, WebSocketAudioClient } from "../services/WebSocketClient";
import { useNotificationStore } from "../stores/notificationStore";
import type { Notification } from "../types/notification";

export const useNotificationIntegration = (wsClient: WebSocketAudioClient | null) => {
  const { pushNotifications } = useNotificationStore();

  // Convert WebSocket message to notification
  const convertToNotification = useCallback(
    (message: ChatMessage): Notification | null => {
      // Handle tool permission notifications
      if (message.metadata?.eventType === "tool_permission") {
        return {
          id: `${message.id}-${Date.now()}`,
          dedupeKey: `tool-permission-${message.metadata.sessionId}-${message.metadata.toolName}`,
          title: "ツール実行許可",
          body: message.text,
          source: "agent",
          sourceId: message.metadata.agentId?.toString() || undefined,
          sourceName: `PID: ${message.metadata.agentId || "Unknown"}`,
          severity: "warning",
          kind: "action_required",
          status: "unread",
          actions: [
            {
              type: "permit",
              label: "許可",
              handler: async () => {
                if (wsClient && message.metadata?.sessionId) {
                  wsClient.sendConfirmResponse("permit", message.id, message.metadata.sessionId);
                }
              },
            },
            {
              type: "deny",
              label: "拒否",
              handler: async () => {
                if (wsClient && message.metadata?.sessionId) {
                  wsClient.sendConfirmResponse("deny", message.id, message.metadata.sessionId);
                }
              },
            },
          ],
          metadata: {
            toolName: message.metadata.toolName,
            sessionId: message.metadata.sessionId,
          },
          createdAt: new Date(message.timestamp),
          sessionId: message.metadata.sessionId || undefined,
        };
      }

      // Handle notification messages only (excluding system messages)
      if (message.type === "notification") {
        return {
          id: `${message.id}-${Date.now()}`,
          title: "通知",
          body: message.text,
          source: "system",
          severity: "warning",
          kind: "action_required",
          status: "unread",
          createdAt: new Date(message.timestamp),
          metadata: message.metadata || undefined,
        };
      }

      return null;
    },
    [wsClient],
  );

  // Listen to WebSocket messages
  useEffect(() => {
    if (!wsClient) return;

    const handleMessage = (message: ChatMessage) => {
      const notification = convertToNotification(message);
      if (notification) {
        pushNotifications([notification]);

        // Play sound if enabled
        const settings = useNotificationStore.getState().settings;
        if (settings.soundEnabled && notification.severity !== "info") {
          // Play notification sound
          const audio = new Audio("/notification.mp3");
          audio.volume = 0.3;
          audio.play().catch(() => {});
        }
      }
    };

    const cleanup = wsClient.addMessageListener(handleMessage);
    return cleanup;
  }, [wsClient, convertToNotification, pushNotifications]);
};
