import {
  Badge,
  Box,
  Button,
  Card,
  Collapse,
  Group,
  ScrollArea,
  Stack,
  Text,
  Title,
} from "@mantine/core";
import {
  IconCheck,
  IconClock,
  IconFolder,
  IconHash,
  IconRefresh,
  IconRobot,
  IconShieldCheck,
  IconX,
} from "@tabler/icons-react";
import type React from "react";
import { useCallback, useEffect, useState } from "react";
import type { Agent } from "../services/AgentService";
import { AgentService } from "../services/AgentService";
import type { ChatMessage, WebSocketAudioClient } from "../services/WebSocketClient";

interface PermissionRequest {
  id: string;
  messageId: string;
  sessionId: string;
  toolName: string;
  text: string;
  timestamp: Date;
}

interface AgentListProps {
  onAgentClick?: (agent: Agent) => void;
  selectedAgentPID?: number | null;
  wsClient?: WebSocketAudioClient | null;
}

export const AgentList: React.FC<AgentListProps> = ({
  onAgentClick,
  selectedAgentPID,
  wsClient,
}) => {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(false);
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date());
  const [permissionsByPID, setPermissionsByPID] = useState<Map<number, PermissionRequest[]>>(
    new Map(),
  );
  const [unknownSessionPermissions, setUnknownSessionPermissions] = useState<PermissionRequest[]>(
    [],
  );
  const [respondedPermissions, setRespondedPermissions] = useState<Set<string>>(new Set());

  const agentService = new AgentService();

  // WebSocketメッセージハンドラー
  const handleWebSocketMessage = useCallback(
    (message: ChatMessage) => {
      // tool_permissionイベントのみ処理
      if (message.metadata?.eventType === "tool_permission") {
        console.log("Tool permission received:", message.metadata);

        const newRequest: PermissionRequest = {
          id: `${message.id}-${Date.now()}`,
          messageId: message.id,
          sessionId: message.metadata.sessionId || "",
          toolName: message.metadata.toolName || "Unknown Tool",
          text: message.text,
          timestamp: new Date(message.timestamp),
        };

        // Check if agentId is provided in the metadata
        const agentId = message.metadata.agentId;
        if (agentId) {
          // Find agent with matching PID
          const matchingAgent = agents.find((agent) => agent.pid === agentId);
          console.log("Matching agent for agentId", agentId, ":", matchingAgent);
          console.log("Current agents:", agents);

          if (matchingAgent) {
            setPermissionsByPID((prev) => {
              const newMap = new Map(prev);
              const pid = agentId;
              const existing = newMap.get(pid) || [];
              newMap.set(pid, [...existing, newRequest]);
              console.log("Added permission request to PID", pid);
              return newMap;
            });
          } else {
            // Add to unknown session permissions
            console.warn("No matching agent found for agentId:", agentId, "Adding to unknown list");
            setUnknownSessionPermissions((prev) => [...prev, newRequest]);
          }
        } else {
          // Fallback to sessionId matching if agentId is not available
          const sessionId = message.metadata.sessionId;
          if (sessionId) {
            const matchingAgent = agents.find((agent) => agent.sessionId === sessionId);
            console.log("Fallback: Matching agent for sessionId", sessionId, ":", matchingAgent);

            if (matchingAgent) {
              setPermissionsByPID((prev) => {
                const newMap = new Map(prev);
                const pid = matchingAgent.pid;
                const existing = newMap.get(pid) || [];
                newMap.set(pid, [...existing, newRequest]);
                console.log("Added permission request to PID", pid);
                return newMap;
              });
            } else {
              console.warn(
                "No matching agent found for sessionId:",
                sessionId,
                "Adding to unknown list",
              );
              setUnknownSessionPermissions((prev) => [...prev, newRequest]);
            }
          }
        }
      }
    },
    [agents],
  );

  // WebSocketメッセージリスナーの登録
  useEffect(() => {
    if (!wsClient) return;

    const cleanup = wsClient.addMessageListener(handleWebSocketMessage);
    return cleanup;
  }, [wsClient, handleWebSocketMessage]);

  // 許可/拒否の処理
  const handlePermissionResponse = (
    pid: number,
    request: PermissionRequest,
    action: "permit" | "deny",
  ) => {
    if (!wsClient) return;

    // Send confirmation response
    wsClient.sendConfirmResponse(action, request.messageId, request.sessionId);

    // Mark as responded
    setRespondedPermissions((prev) => new Set(prev).add(request.id));

    // Remove from pending permissions after a short delay
    setTimeout(() => {
      setPermissionsByPID((prev) => {
        const newMap = new Map(prev);
        const existing = newMap.get(pid) || [];
        newMap.set(
          pid,
          existing.filter((p) => p.id !== request.id),
        );
        if (newMap.get(pid)?.length === 0) {
          newMap.delete(pid);
        }
        return newMap;
      });

      // Also remove from unknown session permissions if it exists
      setUnknownSessionPermissions((prev) => prev.filter((p) => p.id !== request.id));
    }, 300);
  };

  // エージェント一覧を取得
  const fetchAgents = async () => {
    setLoading(true);
    try {
      const fetchedAgents = await agentService.getAgents();
      setAgents(fetchedAgents);
      setLastUpdate(new Date());

      // Check if any unknown session permissions now match fetched agents
      setUnknownSessionPermissions((prev) => {
        const remaining: PermissionRequest[] = [];
        prev.forEach((request) => {
          const matchingAgent = fetchedAgents.find(
            (agent) => agent.sessionId === request.sessionId,
          );
          if (matchingAgent) {
            // Move to the correct agent's permission list
            setPermissionsByPID((pidMap) => {
              const newMap = new Map(pidMap);
              const existing = newMap.get(matchingAgent.pid) || [];
              newMap.set(matchingAgent.pid, [...existing, request]);
              return newMap;
            });
          } else {
            remaining.push(request);
          }
        });
        return remaining;
      });
    } finally {
      setLoading(false);
    }
  };

  // 初回読み込みと定期更新
  useEffect(() => {
    fetchAgents();

    // 5秒ごとに更新
    const interval = setInterval(fetchAgents, 5000);

    return () => clearInterval(interval);
  }, []);

  // 相対時間を計算
  const getRelativeTime = (dateString: string): string => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffSec = Math.floor(diffMs / 1000);
    const diffMin = Math.floor(diffSec / 60);
    const diffHour = Math.floor(diffMin / 60);
    const diffDay = Math.floor(diffHour / 24);

    if (diffDay > 0) {
      return `${diffDay}日前`;
    }
    if (diffHour > 0) {
      return `${diffHour}時間前`;
    }
    if (diffMin > 0) {
      return `${diffMin}分前`;
    }
    return `${diffSec}秒前`;
  };

  return (
    <Box style={{ height: "100%", display: "flex", flexDirection: "column", padding: "16px" }}>
      {/* ヘッダー */}
      <Group justify="space-between" mb="md">
        <Group gap="xs">
          <IconRobot size={24} stroke={1.5} />
          <Title order={4}>Active Agents</Title>
        </Group>
        <Group gap="xs">
          {loading && <IconRefresh size={18} style={{ animation: "spin 1s linear infinite" }} />}
          <Badge variant="light" color="blue">
            {agents.length} agents
          </Badge>
        </Group>
      </Group>

      {/* エージェントリスト */}
      <ScrollArea style={{ flex: 1 }}>
        <Stack gap="sm">
          {agents.length === 0 ? (
            <Card padding="lg" radius="md" withBorder>
              <Text ta="center" c="dimmed">
                エージェントが見つかりません
              </Text>
              <Text ta="center" size="sm" c="dimmed" mt="xs">
                Claude Companionが起動されるのを待っています...
              </Text>
            </Card>
          ) : (
            agents.map((agent) => {
              const permissions = permissionsByPID.get(agent.pid) || [];
              const hasPermissions = permissions.length > 0;

              return (
                <Box key={agent.pid}>
                  <Card
                    padding="sm"
                    radius="md"
                    withBorder
                    style={{
                      cursor: onAgentClick ? "pointer" : "default",
                      backgroundColor:
                        selectedAgentPID === agent.pid ? "var(--mantine-color-blue-9)" : undefined,
                      transition: "background-color 0.2s",
                      borderBottomLeftRadius: hasPermissions ? 0 : undefined,
                      borderBottomRightRadius: hasPermissions ? 0 : undefined,
                    }}
                    onClick={() => onAgentClick?.(agent)}
                  >
                    <Stack gap="xs">
                      {/* プロジェクト名 */}
                      <Group gap="xs" justify="space-between">
                        <Group gap="xs">
                          <IconFolder size={16} stroke={1.5} />
                          <Text fw={600} size="sm">
                            {agent.projectName}
                          </Text>
                        </Group>
                        {hasPermissions && (
                          <Badge color="yellow" variant="filled" size="xs">
                            <IconShieldCheck size={12} /> 許可待ち
                          </Badge>
                        )}
                      </Group>

                      {/* セッションID */}
                      <Group gap="xs">
                        <IconHash size={14} stroke={1.5} style={{ opacity: 0.6 }} />
                        <Text size="xs" c="dimmed" style={{ fontFamily: "monospace" }}>
                          {agent.sessionId}
                        </Text>
                      </Group>

                      {/* PIDと更新時刻 */}
                      <Group justify="space-between">
                        <Badge variant="outline" size="sm">
                          PID: {agent.pid}
                        </Badge>
                        <Group gap={4}>
                          <IconClock size={12} stroke={1.5} style={{ opacity: 0.6 }} />
                          <Text size="xs" c="dimmed">
                            {getRelativeTime(agent.updatedAt)}
                          </Text>
                        </Group>
                      </Group>
                    </Stack>
                  </Card>

                  {/* 許可待ちパネル */}
                  <Collapse in={hasPermissions}>
                    <Card
                      padding="sm"
                      radius="md"
                      withBorder
                      style={{
                        borderTop: "none",
                        borderTopLeftRadius: 0,
                        borderTopRightRadius: 0,
                        backgroundColor: "rgba(255, 200, 0, 0.1)",
                        borderColor: "rgba(255, 200, 0, 0.3)",
                      }}
                      onClick={(e) => e.stopPropagation()}
                    >
                      <Stack gap="xs">
                        {permissions.map((request) => (
                          <Box key={request.id}>
                            <Group gap="xs" mb="xs">
                              <Badge color="orange" variant="light" size="xs">
                                🔧 {request.toolName}
                              </Badge>
                              <Text size="xs" c="dimmed">
                                {request.timestamp.toLocaleTimeString("ja-JP", {
                                  hour: "2-digit",
                                  minute: "2-digit",
                                  second: "2-digit",
                                })}
                              </Text>
                            </Group>
                            <Text size="xs" c="white" lineClamp={2} mb="xs">
                              {request.text}
                            </Text>
                            <Group gap="xs">
                              <Button
                                size="xs"
                                color="green"
                                variant="light"
                                leftSection={<IconCheck size={14} />}
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handlePermissionResponse(agent.pid, request, "permit");
                                }}
                                disabled={respondedPermissions.has(request.id)}
                                style={{ flex: 1 }}
                              >
                                許可
                              </Button>
                              <Button
                                size="xs"
                                color="red"
                                variant="light"
                                leftSection={<IconX size={14} />}
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handlePermissionResponse(agent.pid, request, "deny");
                                }}
                                disabled={respondedPermissions.has(request.id)}
                                style={{ flex: 1 }}
                              >
                                拒否
                              </Button>
                            </Group>
                          </Box>
                        ))}
                      </Stack>
                    </Card>
                  </Collapse>
                </Box>
              );
            })
          )}

          {/* Unknown session permissions */}
          {unknownSessionPermissions.length > 0 && (
            <Card
              padding="sm"
              radius="md"
              withBorder
              style={{
                backgroundColor: "rgba(255, 200, 0, 0.1)",
                borderColor: "rgba(255, 200, 0, 0.3)",
              }}
            >
              <Stack gap="xs">
                <Group gap="xs" justify="space-between">
                  <Text fw={600} size="sm">
                    Unknown Agent
                  </Text>
                  <Badge color="yellow" variant="filled" size="xs">
                    <IconShieldCheck size={12} /> 許可待ち
                  </Badge>
                </Group>
                <Text size="xs" c="dimmed">
                  Session: {unknownSessionPermissions[0]?.sessionId || "N/A"}
                </Text>
                {unknownSessionPermissions.map((request) => (
                  <Box key={request.id}>
                    <Group gap="xs" mb="xs">
                      <Badge color="orange" variant="light" size="xs">
                        🔧 {request.toolName}
                      </Badge>
                      <Text size="xs" c="dimmed">
                        {request.timestamp.toLocaleTimeString("ja-JP", {
                          hour: "2-digit",
                          minute: "2-digit",
                          second: "2-digit",
                        })}
                      </Text>
                    </Group>
                    <Text size="xs" c="white" lineClamp={2} mb="xs">
                      {request.text}
                    </Text>
                    <Group gap="xs">
                      <Button
                        size="xs"
                        color="green"
                        variant="light"
                        leftSection={<IconCheck size={14} />}
                        onClick={() => {
                          handlePermissionResponse(0, request, "permit");
                        }}
                        disabled={respondedPermissions.has(request.id)}
                        style={{ flex: 1 }}
                      >
                        許可
                      </Button>
                      <Button
                        size="xs"
                        color="red"
                        variant="light"
                        leftSection={<IconX size={14} />}
                        onClick={() => {
                          handlePermissionResponse(0, request, "deny");
                        }}
                        disabled={respondedPermissions.has(request.id)}
                        style={{ flex: 1 }}
                      >
                        拒否
                      </Button>
                    </Group>
                  </Box>
                ))}
              </Stack>
            </Card>
          )}
        </Stack>
      </ScrollArea>

      {/* フッター（最終更新時刻） */}
      <Box mt="md" pt="sm" style={{ borderTop: "1px solid var(--mantine-color-gray-7)" }}>
        <Text size="xs" c="gray.5" ta="center">
          最終更新: {lastUpdate.toLocaleTimeString("ja-JP")}
        </Text>
      </Box>
    </Box>
  );
};

// アニメーション用のCSS
const style = document.createElement("style");
style.textContent = `
  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }
`;
document.head.appendChild(style);
