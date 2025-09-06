import type { components } from "../types/api";

// Type aliases for convenience
export type ConnectionStatus = components["schemas"]["ConnectionStatus"];
export type MessageType = components["schemas"]["MessageType"];
export type ChatMessage = components["schemas"]["ChatMessage"];
export type ClientMessage = components["schemas"]["ClientMessage"];
export type WebSocketConnectionState = components["schemas"]["WebSocketConnectionState"];
export type MessageMetadata = components["schemas"]["MessageMetadata"];

// Re-export for backward compatibility
export type MessageRole = ChatMessage["role"];
export type AssistantMessageSubType = ChatMessage["subType"];

export class WebSocketAudioClient {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private readonly maxReconnectAttempts = 10;
  private readonly initialReconnectDelay = 1000; // 初期リトライ間隔: 1秒
  private readonly maxReconnectDelay = 30000; // 最大リトライ間隔: 30秒
  private heartbeatInterval: NodeJS.Timeout | null = null;
  private isConnecting = false;
  private reconnectTimeout: NodeJS.Timeout | null = null;
  private currentState: WebSocketConnectionState = { mode: "timeline" };
  private messageListeners: Set<(message: ChatMessage) => void> = new Set();

  constructor(
    private readonly url: string,
    onMessage: (message: ChatMessage) => void,
    private readonly onStatusChange: (status: ConnectionStatus) => void,
  ) {
    // Add the default message handler
    this.messageListeners.add(onMessage);
  }

  connect(): void {
    // 既存のリトライタイマーをクリア
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }

    // Already connecting or connected
    if (this.isConnecting) {
      return;
    }

    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    if (this.ws?.readyState === WebSocket.CONNECTING) {
      return;
    }

    this.isConnecting = true;
    this.onStatusChange("connecting");

    try {
      this.ws = new WebSocket(this.url);
      this.setupEventHandlers();
    } catch (error) {
      console.error("WebSocket connection error:", error);
      this.handleConnectionError();
    }
  }

  private setupEventHandlers(): void {
    if (!this.ws) return;

    this.ws.onopen = () => {
      this.isConnecting = false;
      this.onStatusChange("connected");
      this.reconnectAttempts = 0;
      this.startHeartbeat();

      // 再接続時に現在の状態を復元
      if (this.currentState.mode === "agent" && this.currentState.agentPid) {
        this.sendStateUpdate(this.currentState);
      }
    };

    this.ws.onmessage = (event) => {
      try {
        const message: ChatMessage = JSON.parse(event.data);

        // Handle different message types
        switch (message.type) {
          case "audio":
          case "text":
          case "system":
          case "user":
          case "assistant":
            // Notify all message listeners
            this.messageListeners.forEach((listener) => {
              listener(message);
            });
            break;
          case "pong":
            // Heartbeat response received
            break;
          default:
          // Unknown message type
        }
      } catch (error) {
        console.error("Failed to parse WebSocket message:", error);
      }
    };

    this.ws.onerror = (error) => {
      console.error("WebSocket error:", error);
      this.onStatusChange("error");
    };

    this.ws.onclose = (event) => {
      this.isConnecting = false;
      this.onStatusChange("disconnected");
      this.stopHeartbeat();

      // Attempt reconnection if not intentionally closed
      // Only reconnect if not a normal closure and not a duplicate connection
      if (event.code !== 1000 && event.code !== 1001) {
        this.attemptReconnect();
      }
    };
  }

  private handleConnectionError(): void {
    this.isConnecting = false;
    this.onStatusChange("error");
    this.attemptReconnect();
  }

  private attemptReconnect(): void {
    // 既存のタイマーをクリア
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }

    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      this.onStatusChange("failed");
      return;
    }

    this.reconnectAttempts++;
    // 指数バックオフ with 上限
    const delay = Math.min(
      this.initialReconnectDelay * 2 ** (this.reconnectAttempts - 1),
      this.maxReconnectDelay,
    );

    // エラー状態を維持（connectingに変わるまで）
    this.onStatusChange("error");

    this.reconnectTimeout = setTimeout(() => {
      this.reconnectTimeout = null;
      this.connect();
    }, delay);
  }

  private startHeartbeat(): void {
    this.stopHeartbeat();

    // Send ping every 30 seconds
    this.heartbeatInterval = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: "ping" }));
      }
    }, 30000);
  }

  private stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }

  disconnect(): void {
    this.stopHeartbeat();

    // リトライタイマーをクリア
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }

    if (this.ws) {
      // Remove event handlers to prevent memory leaks
      this.ws.onopen = null;
      this.ws.onmessage = null;
      this.ws.onerror = null;
      this.ws.onclose = null;

      // Close with normal closure code
      if (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING) {
        this.ws.close(1000, "Client disconnecting");
      }
      this.ws = null;
    }

    // リトライカウンターをリセット
    this.reconnectAttempts = 0;
    this.isConnecting = false;
    this.onStatusChange("disconnected");
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  getReadyState(): number | undefined {
    return this.ws?.readyState;
  }

  // メッセージ送信機能を追加
  sendMessage(text: string, sessionId: string): void {
    if (this.ws?.readyState !== WebSocket.OPEN) {
      return;
    }

    if (!sessionId) {
      return;
    }

    const message: ClientMessage = {
      type: "user_message",
      sessionId: sessionId,
      text: text,
      timestamp: new Date().toISOString(),
    };

    try {
      this.ws.send(JSON.stringify(message));
    } catch (error) {
      console.error("Failed to send message:", error);
    }
  }

  // Send state update to server
  private sendStateUpdate(state: WebSocketConnectionState): void {
    if (this.ws?.readyState !== WebSocket.OPEN) {
      return;
    }

    const message: ClientMessage = {
      type: "update_state",
      state: state,
      timestamp: new Date().toISOString(),
    };

    try {
      this.ws.send(JSON.stringify(message));
    } catch (error) {
      console.error("Failed to send state update:", error);
    }
  }

  // Update connection state
  updateState(state: WebSocketConnectionState): void {
    this.currentState = state;

    // Check connection state
    if (!this.ws) {
      return;
    }

    if (this.ws.readyState === WebSocket.CONNECTING) {
      // Wait for connection to be established
      const checkConnection = setInterval(() => {
        if (this.ws?.readyState === WebSocket.OPEN) {
          clearInterval(checkConnection);
          this.sendStateUpdate(state);
        } else if (
          this.ws?.readyState === WebSocket.CLOSED ||
          this.ws?.readyState === WebSocket.CLOSING
        ) {
          clearInterval(checkConnection);
        }
      }, 100);
      return;
    }

    if (this.ws.readyState !== WebSocket.OPEN) {
      return;
    }

    this.sendStateUpdate(state);
  }

  // Set agent mode for filtering messages
  setAgentMode(agentPID: number): void {
    this.updateState({ mode: "agent", agentPid: agentPID });
  }

  // Clear agent mode and return to timeline mode
  clearAgentMode(): void {
    this.updateState({ mode: "timeline" });
  }

  // Add a message listener
  addMessageListener(listener: (message: ChatMessage) => void): () => void {
    this.messageListeners.add(listener);
    // Return a cleanup function
    return () => {
      this.messageListeners.delete(listener);
    };
  }

  // Remove a message listener
  removeMessageListener(listener: (message: ChatMessage) => void): void {
    this.messageListeners.delete(listener);
  }

  // Send confirmation response for tool permissions
  sendConfirmResponse(action: "permit" | "deny", messageId: string, sessionId: string): void {
    if (this.ws?.readyState !== WebSocket.OPEN) {
      return;
    }

    if (!sessionId) {
      return;
    }

    const message: ClientMessage = {
      type: "confirm_response",
      sessionId: sessionId,
      action: action,
      messageId: messageId,
      timestamp: new Date().toISOString(),
    };

    try {
      this.ws.send(JSON.stringify(message));
    } catch (error) {
      console.error("Failed to send confirm response:", error);
    }
  }
}
