import type { components } from "../types/api";

// TypeSpecから生成された型を使用
export type Agent = components["schemas"]["Agent"];
export type AgentListResponse = components["schemas"]["AgentListResponse"];

// エージェント関連のAPIサービス
export class AgentService {
  private apiUrl: string;

  constructor(apiUrl?: string) {
    // WebSocketと同じホストとポートを使用
    const wsUrl = import.meta.env.VITE_WS_URL || "ws://localhost:8080/ws/audio";
    const baseUrl = wsUrl.replace(/^ws/, "http").replace(/\/ws\/audio$/, "");
    this.apiUrl = apiUrl || `${baseUrl}/api`;
  }

  // エージェント一覧を取得
  async getAgents(): Promise<Agent[]> {
    try {
      const response = await fetch(`${this.apiUrl}/agents`, {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
        },
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch agents: ${response.statusText}`);
      }

      const data: AgentListResponse = await response.json();
      return data.agents;
    } catch (error) {
      console.error("Error fetching agents:", error);
      return [];
    }
  }
}
