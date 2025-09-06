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
  tool_use: "tool_use",
  tool_result: "tool_result",
  content_block: "content_block",
  error: "error",
}

/**
 * Chat message sent via WebSocket
 */
model ChatMessage {
  /**
   * Unique message ID
   */
  id?: string;

  /**
   * Type of the message
   */
  type: MessageType;

  /**
   * Role of the message sender
   */
  role?: "user" | "assistant" | "system";

  /**
   * Sub-type for assistant messages
   */
  subType?: "audio" | "text";

  /**
   * Text content of the message
   */
  text?: string;

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
  timestamp?: string;

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
 * Connection status for WebSocket
 */
enum ConnectionStatus {
  connecting: "connecting",
  connected: "connected",
  disconnected: "disconnected",
  error: "error",
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