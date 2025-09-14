import { Badge, Box, Group, Text, UnstyledButton } from "@mantine/core";
import { IconBell } from "@tabler/icons-react";
import type React from "react";
import { useNotificationStore } from "../../stores/notificationStore";

export const NotificationBar: React.FC = () => {
  const { isOpen, togglePanel, getUnreadCount } = useNotificationStore();
  const unreadCount = getUnreadCount();

  return (
    <UnstyledButton
      onClick={togglePanel}
      aria-expanded={isOpen}
      aria-label={`通知センター ${unreadCount > 0 ? `未読 ${unreadCount} 件` : ""}`}
      style={{
        width: "100%",
        display: "block",
      }}
    >
      <Box
        px="md"
        py="sm"
        style={{
          borderTop: "1px solid var(--mantine-color-gray-8)",
          backgroundColor: isOpen ? "var(--mantine-color-dark-6)" : "var(--mantine-color-dark-7)",
          transition: "background-color 200ms ease",
          cursor: "pointer",
        }}
        onMouseEnter={(e) => {
          if (!isOpen) {
            e.currentTarget.style.backgroundColor = "var(--mantine-color-dark-6)";
          }
        }}
        onMouseLeave={(e) => {
          if (!isOpen) {
            e.currentTarget.style.backgroundColor = "var(--mantine-color-dark-7)";
          }
        }}
      >
        <Group justify="space-between" wrap="nowrap">
          <Group gap="xs" wrap="nowrap">
            <IconBell
              size={18}
              stroke={1.5}
              style={{
                color:
                  unreadCount > 0 ? "var(--mantine-color-yellow-5)" : "var(--mantine-color-gray-5)",
              }}
            />
            <Text size="sm" c="white" fw={500}>
              通知
            </Text>
          </Group>

          <Badge
            size="sm"
            variant={unreadCount > 0 ? "filled" : "outline"}
            color={unreadCount > 0 ? "yellow" : "gray"}
            circle
            style={{
              minWidth: "24px",
              opacity: unreadCount === 0 ? 0.5 : 1,
            }}
          >
            {unreadCount}
          </Badge>
        </Group>
      </Box>
    </UnstyledButton>
  );
};
