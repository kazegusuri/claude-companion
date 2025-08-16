export type ConnectionStatus = "disconnected" | "connecting" | "connected" | "error" | "failed";

export type MessageType = "audio" | "text" | "ping" | "pong" | "error";

export interface AudioMessage {
  type: MessageType;
  id: string;
  text: string;
  audioData?: string; // Base64 encoded WAV data
  priority: number;
  timestamp: string;
  metadata?: {
    eventType: string;
    toolName?: string;
    speaker?: number;
    sampleRate?: number;
    duration?: number;
  };
}

export class WebSocketAudioClient {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private readonly maxReconnectAttempts = 5;
  private readonly reconnectDelay = 1000;
  private heartbeatInterval: NodeJS.Timeout | null = null;
  private isConnecting = false;

  constructor(
    private readonly url: string,
    private readonly onMessage: (message: AudioMessage) => void,
    private readonly onStatusChange: (status: ConnectionStatus) => void,
  ) {}

  connect(): void {
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
        const message: AudioMessage = JSON.parse(event.data);

        // Handle different message types
        switch (message.type) {
          case "audio":
          case "text":
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
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error("Max reconnection attempts reached");
      this.onStatusChange("failed");
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

    console.log(
      `Attempting reconnection ${this.reconnectAttempts}/${this.maxReconnectAttempts} in ${delay}ms`,
    );

    setTimeout(() => {
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

    if (this.ws) {
      // Close with normal closure code
      this.ws.close(1000, "Client disconnecting");
      this.ws = null;
    }

    this.onStatusChange("disconnected");
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  getReadyState(): number | undefined {
    return this.ws?.readyState;
  }
}
