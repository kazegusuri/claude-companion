import { Badge, Box, Card, Group, Indicator, Stack, Text } from "@mantine/core";
import type React from "react";
import { useNotificationStore } from "../../stores/notificationStore";
import type { Notification } from "../../types/notification";

interface NotificationItemProps {
  notification: Notification;
}

export const NotificationItem: React.FC<NotificationItemProps> = ({ notification }) => {
  const { markAsRead } = useNotificationStore();

  const handleView = () => {
    markAsRead([notification.id]);
    // Navigate to message/agent view
    if (notification.actions?.find((a) => a.type === "view")) {
      notification.actions.find((a) => a.type === "view")?.handler?.();
    }
  };

  const getStatusColor = () => {
    if (notification.status === "unread") return "violet";
    if (notification.status === "resolved") return "green";
    if (notification.status === "error") return "red";
    if (notification.severity === "critical") return "red";
    if (notification.severity === "warning") return "yellow";
    return "gray";
  };

  const formatTime = (date: Date | string) => {
    const now = new Date();
    const targetDate = typeof date === "string" ? new Date(date) : date;

    // Invalid date check
    if (isNaN(targetDate.getTime())) {
      return "不明";
    }

    const diff = now.getTime() - targetDate.getTime();
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 1) return "今";
    if (minutes < 60) return `${minutes}分前`;
    if (hours < 24) return `${hours}時間前`;
    return `${days}日前`;
  };

  return (
    <Card
      padding="sm"
      radius="md"
      withBorder
      style={{
        cursor: "pointer",
        backgroundColor:
          notification.status === "unread"
            ? "var(--mantine-color-dark-5)"
            : "var(--mantine-color-dark-6)",
        borderColor: "var(--mantine-color-dark-4)",
        transition: "background-color 0.2s, border-color 0.2s",
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.backgroundColor = "var(--mantine-color-dark-4)";
        e.currentTarget.style.borderColor = "var(--mantine-color-dark-3)";
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.backgroundColor =
          notification.status === "unread"
            ? "var(--mantine-color-dark-5)"
            : "var(--mantine-color-dark-6)";
        e.currentTarget.style.borderColor = "var(--mantine-color-dark-4)";
      }}
      onClick={handleView}
    >
      <Group align="flex-start" gap="sm" wrap="nowrap">
        {/* Status Indicator */}
        <Indicator
          color={getStatusColor()}
          size={8}
          offset={0}
          position="middle-center"
          processing={notification.status === "unread"}
        >
          <Box w={8} h={8} />
        </Indicator>

        {/* Content */}
        <Stack gap="xs" style={{ flex: 1 }}>
          {/* Title and Meta */}
          <Group justify="space-between" wrap="nowrap">
            <Text size="sm" fw={600} c="white" lineClamp={1}>
              {notification.title}
            </Text>
            <Group gap="xs" wrap="nowrap">
              {notification.sourceName && (
                <Badge size="xs" variant="light" color="blue">
                  {notification.sourceName}
                </Badge>
              )}
              <Text size="xs" c="dimmed">
                {formatTime(notification.createdAt)}
              </Text>
            </Group>
          </Group>

          {/* Body */}
          <Text size="xs" c="gray.3" lineClamp={2}>
            {notification.body}
          </Text>
        </Stack>
      </Group>
    </Card>
  );
};
