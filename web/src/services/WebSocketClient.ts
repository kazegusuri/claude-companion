export type ConnectionStatus = "disconnected" | "connecting" | "connected" | "error" | "failed";

export type MessageType =
  | "audio"
  | "text"
  | "ping"
  | "pong"
  | "error"
  | "system"
  | "user"
  | "assistant";

export type MessageRole = "system" | "user" | "assistant";
export type AssistantMessageSubType = "audio" | "text";

export interface ChatMessage {
  type: MessageType;
  id: string;
  role: MessageRole;
  text: string;
  audioData?: string; // Base64 encoded WAV data for audio messages
  subType?: AssistantMessageSubType; // For assistant messages
  priority: number;
  timestamp: string;
  metadata?: {
    eventType: string;
    toolName?: string;
    speaker?: number;
    sampleRate?: number;
    duration?: number;
    sessionId?: string;
    role?: MessageRole;
    subType?: AssistantMessageSubType;
  };
}

export class WebSocketAudioClient {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private readonly maxReconnectAttempts = 10;
  private readonly initialReconnectDelay = 1000; // 初期リトライ間隔: 1秒
  private readonly maxReconnectDelay = 30000; // 最大リトライ間隔: 30秒
  private heartbeatInterval: NodeJS.Timeout | null = null;
  private isConnecting = false;
  private reconnectTimeout: NodeJS.Timeout | null = null;

  constructor(
    private readonly url: string,
    private readonly onMessage: (message: ChatMessage) => void,
    private readonly onStatusChange: (status: ConnectionStatus) => void,
  ) {
    // Constructor body
  }

  connect(): void {
    // 既存のリトライタイマーをクリア
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }

    if (this.isConnecting || this.ws?.readyState === WebSocket.OPEN) {
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
      console.log("WebSocket connected to", this.url);
      this.isConnecting = false;
      this.onStatusChange("connected");
      this.reconnectAttempts = 0;
      this.startHeartbeat();
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
            this.onMessage(message);
            break;
          case "pong":
            // Heartbeat response received
            break;
          default:
            console.warn("Unknown message type:", message.type);
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
      console.log("WebSocket closed:", event.code, event.reason);
      this.isConnecting = false;
      this.onStatusChange("disconnected");
      this.stopHeartbeat();

      // Attempt reconnection if not intentionally closed
      if (event.code !== 1000) {
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
      console.error("Max reconnection attempts reached");
      this.onStatusChange("failed");
      return;
    }

    this.reconnectAttempts++;
    // 指数バックオフ with 上限
    const delay = Math.min(
      this.initialReconnectDelay * Math.pow(2, this.reconnectAttempts - 1),
      this.maxReconnectDelay,
    );

    console.log(
      `Attempting reconnection ${this.reconnectAttempts}/${this.maxReconnectAttempts} in ${delay}ms`,
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
      // Close with normal closure code
      this.ws.close(1000, "Client disconnecting");
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
      console.error("WebSocket is not connected");
      return;
    }

    if (!sessionId) {
      console.error("Session ID not provided");
      return;
    }

    const message = {
      type: "user_message",
      sessionId: sessionId,
      text: text,
      timestamp: new Date().toISOString(),
    };

    try {
      this.ws.send(JSON.stringify(message));
      console.log("Message sent:", message);
    } catch (error) {
      console.error("Failed to send message:", error);
    }
  }

  // Send confirmation response for tool permissions
  sendConfirmResponse(action: "permit" | "deny", messageId: string, sessionId: string): void {
    if (this.ws?.readyState !== WebSocket.OPEN) {
      console.error("WebSocket is not connected");
      return;
    }

    if (!sessionId) {
      console.error("Session ID not provided");
      return;
    }

    const message = {
      type: "confirm_response",
      sessionId: sessionId,
      action: action,
      messageId: messageId,
      timestamp: new Date().toISOString(),
    };

    try {
      this.ws.send(JSON.stringify(message));
      console.log("Confirm response sent:", message);
    } catch (error) {
      console.error("Failed to send confirm response:", error);
    }
  }
}
