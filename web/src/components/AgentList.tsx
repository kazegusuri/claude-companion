import { Badge, Box, Card, Group, ScrollArea, Stack, Text, Title } from "@mantine/core";
import { IconClock, IconFolder, IconHash, IconRefresh, IconRobot } from "@tabler/icons-react";
import type React from "react";
import { useEffect, useState } from "react";
import type { Agent } from "../services/AgentService";
import { AgentService } from "../services/AgentService";

export const AgentList: React.FC = () => {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(false);
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date());

  const agentService = new AgentService();

  // エージェント一覧を取得
  const fetchAgents = async () => {
    setLoading(true);
    try {
      const fetchedAgents = await agentService.getAgents();
      setAgents(fetchedAgents);
      setLastUpdate(new Date());
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
            agents.map((agent) => (
              <Card key={agent.pid} padding="sm" radius="md" withBorder>
                <Stack gap="xs">
                  {/* プロジェクト名 */}
                  <Group gap="xs">
                    <IconFolder size={16} stroke={1.5} />
                    <Text fw={600} size="sm">
                      {agent.projectName}
                    </Text>
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
            ))
          )}
        </Stack>
      </ScrollArea>

      {/* フッター（最終更新時刻） */}
      <Box mt="md" pt="sm" style={{ borderTop: "1px solid var(--mantine-color-gray-8)" }}>
        <Text size="xs" c="dimmed" ta="center">
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
