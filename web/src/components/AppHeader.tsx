import { ActionIcon, Box, Button, Group, Tabs, Text, Tooltip } from "@mantine/core";
import { IconDashboard, IconMicrophone, IconRobot } from "@tabler/icons-react";
import type React from "react";
import { resumeSharedAudioContext } from "../utils/live2d/audioContextPatch";

interface AppHeaderProps {
  currentView: "dashboard" | "narrator" | "live2d";
  onViewChange: (view: "dashboard" | "narrator" | "live2d") => void;
  isAudioEnabled: boolean;
  onAudioToggle: (enabled: boolean) => void;
}

export const AppHeader: React.FC<AppHeaderProps> = ({
  currentView,
  onViewChange,
  isAudioEnabled,
  onAudioToggle,
}) => {
  const handleToggleAudio = async () => {
    if (!isAudioEnabled) {
      onAudioToggle(true);
      // 共有AudioContextをresume（PWA対策）
      await resumeSharedAudioContext();
    } else {
      onAudioToggle(false);
    }
  };
  return (
    <Box
      component="header"
      style={{
        height: "100%", // AppShell.Header の高さに合わせる
        backgroundColor: "var(--mantine-color-dark-7)",
        borderBottom: "1px solid var(--mantine-color-dark-5)",
        padding: "0 20px",
        display: "flex",
        alignItems: "center",
      }}
    >
      <Group justify="space-between" style={{ width: "100%" }}>
        <Group>
          <Text size="xl" fw={700} c="white">
            Claude Companion
          </Text>
          <Text size="sm" c="dimmed">
            Web Interface
          </Text>
        </Group>

        <Tabs
          value={currentView}
          onChange={(value) => onViewChange(value as "dashboard" | "narrator" | "live2d")}
          variant="pills"
          radius="lg"
        >
          <Tabs.List>
            <Tabs.Tab value="dashboard" leftSection={<IconDashboard size={16} />}>
              Dashboard
            </Tabs.Tab>
            <Tabs.Tab value="narrator" leftSection={<IconMicrophone size={16} />}>
              Audio Narrator
            </Tabs.Tab>
            <Tabs.Tab value="live2d" leftSection={<IconRobot size={16} />}>
              Live2D Viewer
            </Tabs.Tab>
          </Tabs.List>
        </Tabs>

        <Group gap="xs">
          <Tooltip label={isAudioEnabled ? "音声ON" : "音声OFF"} position="bottom" withArrow>
            <ActionIcon
              onClick={handleToggleAudio}
              size="md"
              radius="xl"
              variant={isAudioEnabled ? "filled" : "light"}
              color={isAudioEnabled ? "green" : "gray"}
            >
              {isAudioEnabled ? "🔊" : "🔇"}
            </ActionIcon>
          </Tooltip>
          <Button variant="subtle" size="sm">
            Settings
          </Button>
          <Button variant="light" size="sm">
            Connect
          </Button>
        </Group>
      </Group>
    </Box>
  );
};
