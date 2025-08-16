import React, { useEffect, useRef, useState, useCallback } from "react";
import { WebSocketAudioClient } from "../services/WebSocketClient";
import type { ConnectionStatus, AudioMessage } from "../services/WebSocketClient";
import { AudioPlayer } from "../services/AudioPlayer";
import "./AudioNarrator.css";

interface MessageHistory {
  id: string;
  text: string;
  timestamp: Date;
  metadata?: AudioMessage["metadata"];
  isPlaying?: boolean;
}

export const AudioNarrator: React.FC = () => {
  const [messages, setMessages] = useState<MessageHistory[]>([]);
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>("disconnected");
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentMessageId, setCurrentMessageId] = useState<string | null>(null);
  const [volume, setVolume] = useState(1.0);
  const [audioQueue, setAudioQueue] = useState<AudioMessage[]>([]);
  const [isAudioEnabled, setIsAudioEnabled] = useState(false); // Global audio enable/disable state

  const wsClient = useRef<WebSocketAudioClient | null>(null);
  const audioPlayer = useRef<AudioPlayer | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const isProcessingQueue = useRef(false);

  // Initialize services
  useEffect(() => {
    // Clean up any existing connections first
    if (wsClient.current) {
      wsClient.current.disconnect();
      wsClient.current = null;
    }
    if (audioPlayer.current) {
      audioPlayer.current.stop();
      audioPlayer.current = null;
    }

    // Create WebSocket client
    const wsUrl = import.meta.env.VITE_WS_URL || "ws://localhost:8080/ws/audio";
    wsClient.current = new WebSocketAudioClient(wsUrl, handleWebSocketMessage, setConnectionStatus);

    // Don't create audio player here - create it when needed

    // Connect to WebSocket
    wsClient.current.connect();

    // Cleanup on unmount
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
  }, []); // Remove isAudioInitialized from dependencies to prevent re-initialization

  // Handle volume changes
  useEffect(() => {
    audioPlayer.current?.setVolume(volume);
  }, [volume]);

  // Process audio queue
  useEffect(() => {
    if (audioQueue.length > 0 && !isProcessingQueue.current) {
      processNextInQueue();
    }
  }, [audioQueue]);

  const handleWebSocketMessage = useCallback((message: AudioMessage) => {
    // Add to message history (check for duplicates)
    setMessages((prev) => {
      // Check if message with this ID already exists
      if (prev.some((msg) => msg.id === message.id)) {
        return prev;
      }

      const historyItem: MessageHistory = {
        id: message.id,
        text: message.text,
        timestamp: new Date(message.timestamp),
        metadata: message.metadata,
      };

      return [...prev, historyItem];
    });

    // Add to audio queue if it contains audio data
    if (message.type === "audio" && message.audioData) {
      setAudioQueue((prev) => {
        // Check if message with this ID already exists in queue
        if (prev.some((msg) => msg.id === message.id)) {
          return prev;
        }

        // Sort by priority (higher priority first)
        const newQueue = [...prev, message];
        newQueue.sort((a, b) => b.priority - a.priority);
        return newQueue;
      });
    }

    // Auto-scroll to bottom
    setTimeout(() => {
      messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }, 100);
  }, []);

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
        audioPlayer.current.setVolume(volume);
      }
      setCurrentMessageId(message.id);
      setIsPlaying(true);

      // Update message to show it's playing
      setMessages((prev) =>
        prev.map((msg) => (msg.id === message.id ? { ...msg, isPlaying: true } : msg)),
      );

      try {
        await audioPlayer.current.playBase64Audio(message.audioData, {
          onEnd: () => {
            // Remove from queue
            setAudioQueue((prev) => prev.slice(1));

            // Update UI
            setIsPlaying(false);
            setCurrentMessageId(null);
            setMessages((prev) =>
              prev.map((msg) => (msg.id === message.id ? { ...msg, isPlaying: false } : msg)),
            );

            isProcessingQueue.current = false;

            // Process next item if available
            if (audioQueue.length > 1) {
              setTimeout(processNextInQueue, 100);
            }
          },
          onError: (error) => {
            console.warn("Audio playback skipped:", error.message);
            // Remove failed item from queue
            setAudioQueue((prev) => prev.slice(1));
            setMessages((prev) =>
              prev.map((msg) => (msg.id === message.id ? { ...msg, isPlaying: false } : msg)),
            );
            isProcessingQueue.current = false;
            setIsPlaying(false);
            setCurrentMessageId(null);

            // Process next item if available
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
      // No audio data, remove from queue
      setAudioQueue((prev) => prev.slice(1));
      isProcessingQueue.current = false;
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

  const handleToggleAudio = async () => {
    if (!isAudioEnabled) {
      // Try to enable audio
      try {
        // Create audio player if not exists
        if (!audioPlayer.current) {
          audioPlayer.current = new AudioPlayer();
          audioPlayer.current.setVolume(volume);
        }
        // Try to initialize AudioContext
        await audioPlayer.current.ensureInitialized();

        // Check if AudioContext is suspended
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
      // Disable audio
      setIsAudioEnabled(false);
      // Stop current playback and clear queue
      audioPlayer.current?.stop();
      setAudioQueue([]);
      setIsPlaying(false);
      setCurrentMessageId(null);
      console.log("Audio disabled");
    }
  };

  return (
    <div className="audio-narrator">
      <div className="narrator-header">
        <h2>Audio Narrator</h2>
        <ConnectionIndicator status={connectionStatus} onReconnect={handleReconnect} />
      </div>

      <div className="message-container">
        {messages.map((message) => (
          <MessageItem
            key={message.id}
            message={message}
            isCurrentlyPlaying={message.id === currentMessageId}
          />
        ))}
        <div ref={messagesEndRef} />
      </div>

      <div className="narrator-controls">
        <div className="playback-controls">
          <button
            onClick={handleToggleAudio}
            className={`control-button toggle-audio-button ${isAudioEnabled ? "enabled" : "disabled"}`}
          >
            {isAudioEnabled ? "🔊 音声ON" : "🔇 音声OFF"}
          </button>
          <button onClick={handleStop} disabled={!isPlaying} className="control-button stop-button">
            {isPlaying ? "■ 停止" : "■ 停止"}
          </button>
          <button onClick={handleClearHistory} className="control-button clear-button">
            履歴クリア
          </button>
        </div>

        <div className="volume-control">
          <label htmlFor="volume">音量:</label>
          <input
            id="volume"
            type="range"
            min="0"
            max="1"
            step="0.1"
            value={volume}
            onChange={(e) => setVolume(parseFloat(e.target.value))}
          />
          <span>{Math.round(volume * 100)}%</span>
        </div>

        {audioQueue.length > 0 && <div className="queue-status">キュー: {audioQueue.length}件</div>}
      </div>
    </div>
  );
};

interface ConnectionIndicatorProps {
  status: ConnectionStatus;
  onReconnect?: () => void;
}

const ConnectionIndicator: React.FC<ConnectionIndicatorProps> = ({ status, onReconnect }) => {
  const getStatusColor = () => {
    switch (status) {
      case "connected":
        return "#4caf50";
      case "connecting":
        return "#ff9800";
      case "error":
      case "failed":
        return "#f44336";
      default:
        return "#9e9e9e";
    }
  };

  const getStatusText = () => {
    switch (status) {
      case "connected":
        return "接続中";
      case "connecting":
        return "接続中...";
      case "disconnected":
        return "切断";
      case "error":
        return "エラー";
      case "failed":
        return "接続失敗";
      default:
        return status;
    }
  };

  return (
    <div className="connection-indicator">
      <span className="status-dot" style={{ backgroundColor: getStatusColor() }} />
      <span className="status-text">{getStatusText()}</span>
      {(status === "disconnected" || status === "failed") && onReconnect && (
        <button onClick={onReconnect} className="reconnect-button">
          再接続
        </button>
      )}
    </div>
  );
};

interface MessageItemProps {
  message: MessageHistory;
  isCurrentlyPlaying: boolean;
}

const MessageItem: React.FC<MessageItemProps> = ({ message, isCurrentlyPlaying }) => {
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

  return (
    <div className={`message-item ${isCurrentlyPlaying ? "playing" : ""}`}>
      <div className="message-header">
        <span className="message-time">{formatTime(message.timestamp)}</span>
        {message.metadata?.eventType && (
          <span className="event-type">
            {getEventTypeIcon(message.metadata.eventType)}
            {message.metadata.eventType}
          </span>
        )}
        {message.metadata?.toolName && (
          <span className="tool-name">{message.metadata.toolName}</span>
        )}
        {isCurrentlyPlaying && (
          <span className="playing-indicator">
            <span className="pulse">●</span> 再生中
          </span>
        )}
      </div>
      <div className="message-text">{message.text}</div>
    </div>
  );
};
