import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

/**
 * Claude Companion API
 * API for managing Claude agent processes and WebSocket communication
 */
@service(#{
  title: "Claude Companion API",
})
@server("http://localhost:8080", "Development server")
namespace ClaudeCompanion;

/**
 * Agent information representing a Claude agent process
 */
model Agent {
  /**
   * Process ID of the agent
   */
  pid: int32;

  /**
   * Session ID of the Claude session
   */
  sessionId: string;

  /**
   * Project directory path
   */
  projectDir: string;

  /**
   * Project name (extracted from projectDir)
   */
  projectName: string;

  /**
   * Timestamp when the agent was created
   */
  @format("date-time")
  createdAt: string;

  /**
   * Timestamp when the agent was last updated
   */
  @format("date-time")
  updatedAt: string;
}

/**
 * Response containing a list of agents
 */
model AgentListResponse {
  /**
   * List of active agents
   */
  agents: Agent[];
}

/**
 * Error response
 */
model ErrorResponse {
  /**
   * Error message
   */
  message: string;
}

/**
 * WebSocket message types
 */
enum MessageType {
  user: "user",
  assistant: "assistant",
  system: "system",
  audio: "audio",
  text: "text",
  ping: "ping",
  pong: "pong",
  user_message: "user_message",
  confirm_response: "confirm_response",
  update_state: "update_state",
  tool_use: "tool_use",
  tool_result: "tool_result",
  content_block: "content_block",
  error: "error",
}

/**
 * Message metadata
 */
model MessageMetadata {
  /**
   * Event type
   */
  eventType: string;

  /**
   * Tool name
   */
  toolName?: string;

  /**
   * Speaker number
   */
  speaker?: int32;

  /**
   * Sample rate for audio
   */
  sampleRate?: int32;

  /**
   * Duration in seconds
   */
  duration?: float32;

  /**
   * Session ID
   */
  sessionId?: string;

  /**
   * Role of the message
   */
  role?: "user" | "assistant" | "system";

  /**
   * Sub-type for assistant messages
   */
  subType?: "audio" | "text";
}

/**
 * WebSocket connection state
 */
model WebSocketConnectionState {
  /**
   * Connection mode
   */
  mode: "timeline" | "agent";

  /**
   * Agent PID when in agent mode
   */
  agentPid?: int32;
}

/**
 * Chat message sent via WebSocket
 */
model ChatMessage {
  /**
   * Unique message ID
   */
  id: string;

  /**
   * Type of the message
   */
  type: MessageType;

  /**
   * Role of the message sender
   */
  role: "user" | "assistant" | "system";

  /**
   * Sub-type for assistant messages
   */
  subType?: "audio" | "text";

  /**
   * Text content of the message
   */
  text: string;

  /**
   * Base64 encoded audio data (WAV format)
   */
  audioData?: string;

  /**
   * Priority for audio playback (higher = higher priority)
   */
  @minValue(0)
  @maxValue(100)
  priority: int32 = 50;

  /**
   * Timestamp of the message
   */
  @format("date-time")
  timestamp: string;

  /**
   * Message metadata
   */
  metadata?: MessageMetadata;

  /**
   * Tool name (for tool_use messages)
   */
  toolName?: string;

  /**
   * Tool input (for tool_use messages)
   */
  toolInput?: Record<unknown>;

  /**
   * Tool result output (for tool_result messages)
   */
  toolOutput?: string;

  /**
   * Error details (for error messages)
   */
  error?: string;

  /**
   * Content details (for content_block messages)
   */
  content?: string;
}

/**
 * Client message sent to WebSocket
 */
model ClientMessage {
  /**
   * Message type
   */
  type: MessageType;

  /**
   * Session ID for the message
   */
  sessionId?: string;

  /**
   * Text content (for user_message)
   */
  text?: string;

  /**
   * Action for confirm_response
   */
  action?: "permit" | "deny";

  /**
   * Message ID for confirm_response
   */
  messageId?: string;

  /**
   * State for update_state
   */
  state?: WebSocketConnectionState;

  /**
   * Timestamp
   */
  @format("date-time")
  timestamp: string;
}

/**
 * Connection status for WebSocket
 */
enum ConnectionStatus {
  connecting: "connecting",
  connected: "connected",
  disconnected: "disconnected",
  error: "error",
  failed: "failed",
}

/**
 * WebSocket status message
 */
model WebSocketStatus {
  /**
   * Current connection status
   */
  status: ConnectionStatus;

  /**
   * Optional error message
   */
  error?: string;

  /**
   * Timestamp of status change
   */
  @format("date-time")
  timestamp: string;
}

/**
 * Agent management API
 */
@route("/api/agents")
@tag("Agents")
interface Agents {
  /**
   * Get list of all active agents
   */
  @get
  list(): AgentListResponse;

  /**
   * Get a specific agent by PID
   */
  @get
  @route("{pid}")
  read(@path pid: int32): Agent | {
    @statusCode statusCode: 404;
    @body error: ErrorResponse;
  };
}