import {
  ActionIcon,
  Box,
  Button,
  Collapse,
  Group,
  ScrollArea,
  SegmentedControl,
  Stack,
  Text,
  Tooltip,
} from "@mantine/core";
import { IconTrash } from "@tabler/icons-react";
import type React from "react";
import { useId } from "react";
import { useNotificationStore } from "../../stores/notificationStore";
import type { NotificationKind } from "../../types/notification";
import { NotificationItem } from "./NotificationItem";

export const NotificationPanel: React.FC = () => {
  const panelId = useId();
  const { isOpen, filter, setFilter, getFilteredNotifications, clearAll, notifications } =
    useNotificationStore();

  const filteredNotifications = getFilteredNotifications();

  const handleFilterChange = (value: string) => {
    setFilter({
      ...filter,
      kind: value as NotificationKind | "all",
    });
  };

  return (
    <Collapse in={isOpen} transitionDuration={200} transitionTimingFunction="ease" id={panelId}>
      <Box
        style={{
          maxHeight: "40vh",
          display: "flex",
          flexDirection: "column",
          backgroundColor: "var(--mantine-color-dark-6)",
          borderTop: "1px solid var(--mantine-color-gray-8)",
        }}
      >
        {/* Toolbar */}
        <Box px="md" py="sm" style={{ borderBottom: "1px solid var(--mantine-color-gray-8)" }}>
          <Stack gap="sm">
            {/* Filter Controls */}
            <Group justify="space-between">
              <SegmentedControl
                size="xs"
                value={filter.kind || "all"}
                onChange={handleFilterChange}
                data={[
                  { label: "すべて", value: "all" },
                  { label: "アクション", value: "action_required" },
                  { label: "情報", value: "info" },
                ]}
              />

              <Tooltip label="すべて削除">
                <ActionIcon
                  variant="subtle"
                  size="sm"
                  onClick={clearAll}
                  disabled={notifications.length === 0}
                >
                  <IconTrash size={16} />
                </ActionIcon>
              </Tooltip>
            </Group>
          </Stack>
        </Box>

        {/* Notification List */}
        <ScrollArea style={{ flex: 1 }} scrollbarSize={6} offsetScrollbars>
          <Box px="md" py="sm">
            <Stack gap="xs">
              {filteredNotifications.length === 0 ? (
                <Box py="xl">
                  <Text size="sm" c="dimmed" ta="center">
                    通知はありません
                  </Text>
                </Box>
              ) : (
                filteredNotifications.map((notification) => (
                  <NotificationItem key={notification.id} notification={notification} />
                ))
              )}
            </Stack>
          </Box>
        </ScrollArea>

        {/* Load More */}
        {filteredNotifications.length >= 20 && (
          <Box px="md" py="xs" style={{ borderTop: "1px solid var(--mantine-color-gray-8)" }}>
            <Button variant="subtle" size="xs" fullWidth disabled>
              さらに読み込む
            </Button>
          </Box>
        )}
      </Box>
    </Collapse>
  );
};
