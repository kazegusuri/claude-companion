import React from "react";
import { Group, Button, Text, Box, Tabs } from "@mantine/core";
import { IconDashboard, IconMicrophone, IconRobot } from "@tabler/icons-react";

interface AppHeaderProps {
  currentView: "dashboard" | "narrator" | "live2d";
  onViewChange: (view: "dashboard" | "narrator" | "live2d") => void;
}

export const AppHeader: React.FC<AppHeaderProps> = ({ currentView, onViewChange }) => {
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
          size="sm"
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
