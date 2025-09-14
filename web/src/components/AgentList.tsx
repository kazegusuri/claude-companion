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
  Tooltip,
} from "@mantine/core";
import {
  IconCheck,
  IconCheckbox,
  IconClock,
  IconFolder,
  IconHash,
  IconRefresh,
  IconRobot,
  IconShieldCheck,
  IconTerminal,
  IconTool,
  IconX,
} from "@tabler/icons-react";
import type React from "react";
import { useCallback, useEffect, useState } from "react";
import type { Agent } from "../services/AgentService";
import { AgentService } from "../services/AgentService";
import type { ChatMessage, WebSocketAudioClient } from "../services/WebSocketClient";

interface AgentListProps {
  onAgentClick?: ((agent: Agent) => void) | undefined;
  selectedAgentPID?: number | null | undefined;
  wsClient?: WebSocketAudioClient | null | undefined;
}

export const AgentList: React.FC<AgentListProps> = ({
  onAgentClick,
  selectedAgentPID,
  wsClient,
}) => {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(false);

  const agentService = new AgentService();

  // エージェント一覧を取得
  const fetchAgents = async () => {
    setLoading(true);
    try {
      const fetchedAgents = await agentService.getAgents();
      setAgents(fetchedAgents);
    } finally {
      setLoading(false);
    }
  };

  // WebSocketメッセージハンドラー
  const handleWebSocketMessage = useCallback((message: ChatMessage) => {
    // tool_permissionイベントを受け取ったら、エージェント情報を再取得
    if (message.metadata?.eventType === "tool_permission") {
      console.log("Tool permission event received, refetching agents...");
      fetchAgents();
    }
  }, []);

  // WebSocketメッセージリスナーの登録
  useEffect(() => {
    if (!wsClient) return;

    const cleanup = wsClient.addMessageListener(handleWebSocketMessage);
    return cleanup;
  }, [wsClient, handleWebSocketMessage]);

  // 許可/拒否の処理
  const handlePermissionResponse = (agent: Agent, action: "permit" | "deny") => {
    if (!wsClient || !agent.session?.activeTool) return;

    // Send confirmation response
    // tool_permissionイベントのmessageIdを使う必要があるが、現在のAPIではmessageIdがない
    // そのため、sessionIdとtoolUseIdから作成する
    const messageId = agent.session.activeTool.toolUseId;
    wsClient.sendConfirmResponse(action, messageId, agent.sessionId);

    // エージェント情報を再取得して状態を更新
    setTimeout(() => {
      fetchAgents();
    }, 300);
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

  // ツールステータスの表示用情報を取得
  const getToolStatusInfo = (tool: { status: string; isError?: boolean; isRejected?: boolean }) => {
    if (tool.status === "running") {
      return { color: "blue", label: "実行中" };
    }
    if (tool.status === "waiting_approval") {
      return { color: "yellow", label: "承認待ち" };
    }
    if (tool.status === "finished") {
      if (tool.isError) {
        return { color: "red", label: "エラー" };
      }
      if (tool.isRejected) {
        return { color: "orange", label: "拒否" };
      }
      return { color: "green", label: "完了" };
    }
    if (tool.status === "created") {
      return { color: "blue", label: "実行中" };
    }
    return { color: "gray", label: tool.status };
  };

  return (
    <Box
      style={{
        height: "100%",
        display: "flex",
        flexDirection: "column",
        backgroundColor: "transparent",
      }}
    >
      {/* ヘッダー */}
      <Group justify="space-between" mb="md" px="md" pt="md">
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
      <ScrollArea style={{ flex: 1 }} px="md">
        <Stack gap="sm" pb="sm">
          {agents.length === 0 ? (
            <Card
              padding="lg"
              radius="md"
              withBorder
              style={{
                backgroundColor: "var(--mantine-color-dark-6)",
                borderColor: "var(--mantine-color-dark-4)",
              }}
            >
              <Text ta="center" c="dimmed">
                エージェントが見つかりません
              </Text>
              <Text ta="center" size="sm" c="dimmed" mt="xs">
                Claude Companionが起動されるのを待っています...
              </Text>
            </Card>
          ) : (
            agents.map((agent) => {
              const hasPermissions = agent.session?.activeTool?.isWaitingApproval === true;

              return (
                <Box key={agent.pid}>
                  <Card
                    padding="sm"
                    radius="md"
                    withBorder
                    style={{
                      cursor: onAgentClick ? "pointer" : "default",
                      backgroundColor:
                        selectedAgentPID === agent.pid
                          ? "var(--mantine-color-blue-9)"
                          : "var(--mantine-color-dark-6)",
                      transition: "background-color 0.2s",
                      borderBottomLeftRadius: hasPermissions ? 0 : undefined,
                      borderBottomRightRadius: hasPermissions ? 0 : undefined,
                      borderColor: "var(--mantine-color-dark-4)",
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

                      {/* Session情報: アクティブツール */}
                      {agent.session?.activeTool &&
                        (() => {
                          const statusInfo = getToolStatusInfo(agent.session.activeTool);
                          return (
                            <Group gap="xs">
                              <IconTool size={14} stroke={1.5} style={{ opacity: 0.8 }} />
                              <Text size="xs" c="dimmed">
                                Active:
                              </Text>
                              <Badge size="xs" variant="light" color={statusInfo.color}>
                                {agent.session.activeTool.toolName} ({statusInfo.label})
                              </Badge>
                            </Group>
                          );
                        })()}

                      {/* Session情報: アクティブタスク */}
                      {agent.session?.activeTasks && agent.session.activeTasks.length > 0 && (
                        <Group gap="xs">
                          <IconCheckbox size={14} stroke={1.5} style={{ opacity: 0.8 }} />
                          <Text size="xs" c="dimmed">
                            Tasks:
                          </Text>
                          <Group gap={4}>
                            {agent.session.activeTasks.slice(0, 3).map((task) => {
                              const getTaskColor = () => {
                                switch (task.status) {
                                  case "completed":
                                    return "green";
                                  case "in_progress":
                                    return "blue";
                                  case "pending":
                                    return "gray";
                                  default:
                                    return "gray";
                                }
                              };
                              const getTaskIcon = () => {
                                switch (task.status) {
                                  case "completed":
                                    return "✓";
                                  case "in_progress":
                                    return "▶";
                                  case "pending":
                                    return "○";
                                  default:
                                    return "";
                                }
                              };
                              return (
                                <Tooltip
                                  key={task.taskId}
                                  label={`${task.taskName}: ${task.description}`}
                                  position="top"
                                >
                                  <Badge size="xs" variant="light" color={getTaskColor()}>
                                    {getTaskIcon()} {task.taskName}
                                  </Badge>
                                </Tooltip>
                              );
                            })}
                            {agent.session.activeTasks.length > 3 && (
                              <Badge size="xs" variant="light" color="gray">
                                +{agent.session.activeTasks.length - 3}
                              </Badge>
                            )}
                          </Group>
                        </Group>
                      )}

                      {/* Session情報: バックグラウンドタスク */}
                      {agent.session?.backgroundTasks &&
                        agent.session.backgroundTasks.length > 0 && (
                          <Group gap="xs">
                            <IconTerminal size={14} stroke={1.5} style={{ opacity: 0.8 }} />
                            <Text size="xs" c="dimmed">
                              Background:
                            </Text>
                            <Group gap={4}>
                              {agent.session.backgroundTasks
                                .filter((task) => !task.isTerminated)
                                .slice(0, 3)
                                .map((task, index) => (
                                  <Tooltip
                                    key={task.backgroundTaskId}
                                    label={task.command || "Unknown command"}
                                    position="top"
                                  >
                                    <Badge size="xs" variant="dot" color="blue">
                                      Task {index + 1}
                                    </Badge>
                                  </Tooltip>
                                ))}
                              {agent.session.backgroundTasks.filter((task) => !task.isTerminated)
                                .length > 3 && (
                                <Badge size="xs" variant="light" color="gray">
                                  +
                                  {agent.session.backgroundTasks.filter(
                                    (task) => !task.isTerminated,
                                  ).length - 3}
                                </Badge>
                              )}
                            </Group>
                          </Group>
                        )}
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
                        <Box>
                          <Group gap="xs" mb="xs">
                            <Badge color="orange" variant="light" size="xs">
                              🔧 {agent.session?.activeTool?.toolName || "Unknown Tool"}
                            </Badge>
                            <Text size="xs" c="dimmed">
                              {new Date(
                                agent.session?.activeTool?.createdAt || "",
                              ).toLocaleTimeString("ja-JP", {
                                hour: "2-digit",
                                minute: "2-digit",
                                second: "2-digit",
                              })}
                            </Text>
                          </Group>
                          <Text size="xs" c="white" mb="xs">
                            ツールの実行許可が必要です
                          </Text>
                          <Group gap="xs">
                            <Button
                              size="xs"
                              color="green"
                              variant="light"
                              leftSection={<IconCheck size={14} />}
                              onClick={(e) => {
                                e.stopPropagation();
                                handlePermissionResponse(agent, "permit");
                              }}
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
                                handlePermissionResponse(agent, "deny");
                              }}
                              style={{ flex: 1 }}
                            >
                              拒否
                            </Button>
                          </Group>
                        </Box>
                      </Stack>
                    </Card>
                  </Collapse>
                </Box>
              );
            })
          )}
        </Stack>
      </ScrollArea>
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
