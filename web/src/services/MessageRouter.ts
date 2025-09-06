import type { Agent } from "./AgentService";
import type { ChatMessage } from "./WebSocketClient";

export type MessageMode = "timeline" | "agent";

export interface MessageRouterState {
  mode: MessageMode;
  selectedAgent: Agent | null;
}

export class MessageRouter {
  private state: MessageRouterState = {
    mode: "timeline",
    selectedAgent: null,
  };

  /**
   * Update the router state
   */
  setState(state: Partial<MessageRouterState>): void {
    this.state = { ...this.state, ...state };
  }

  /**
   * Get current state
   */
  getState(): MessageRouterState {
    return { ...this.state };
  }

  /**
   * Check if a message should be accepted based on current routing rules
   */
  shouldAcceptMessage(message: ChatMessage): boolean {
    // Always accept system messages
    if (message.type === "system") {
      return true;
    }

    // In timeline mode, accept all messages
    if (this.state.mode === "timeline") {
      return true;
    }

    // In agent mode, filter by agent ID
    if (this.state.mode === "agent" && this.state.selectedAgent) {
      const messageAgentId = message.metadata?.agentId;
      const selectedAgentPid = this.state.selectedAgent.pid;

      // Only accept messages from the selected agent
      return messageAgentId === selectedAgentPid;
    }

    // Default to accepting the message
    return true;
  }

  /**
   * Set timeline mode
   */
  setTimelineMode(): void {
    this.setState({ mode: "timeline", selectedAgent: null });
  }

  /**
   * Set agent mode with the selected agent
   */
  setAgentMode(agent: Agent): void {
    this.setState({ mode: "agent", selectedAgent: agent });
  }

  /**
   * Clear current mode and return to timeline
   */
  clearMode(): void {
    this.setTimelineMode();
  }
}

// Create a singleton instance
export const messageRouter = new MessageRouter();
