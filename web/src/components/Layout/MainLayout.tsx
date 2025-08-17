import React from "react";
import { Paper, Grid, Card, Title, Text, Stack, Box, ScrollArea, Group } from "@mantine/core";
import "./MainLayout.css";

interface MainLayoutProps {
  modelComponent?: React.ReactNode;
  scheduleComponent?: React.ReactNode;
  textComponent?: React.ReactNode;
  chatComponent?: React.ReactNode;
}

const HEADER = 60; // AppShellのheader高さ
const ModelHeight = "400px";
const ScheduleHeight = "clamp(260px, 35dvh, 420px)";

export const MainLayout: React.FC<MainLayoutProps> = ({
  modelComponent,
  scheduleComponent,
  textComponent,
  chatComponent,
}) => {
  return (
    <Box
      className="main-layout"
      style={{
        display: "flex",
        flexDirection: "column",
        flex: 1,
        minHeight: 0,
        overflow: "hidden",
      }}
    >
      {/* 画面高を確保（ヘッダー＆パディングを差し引く）
          重要: 100dvh を使用（モバイルのアドレスバー対策）
          重要: align="stretch" で左右カラムの高さを自動的に揃える
          横幅制限: App.tsxの AppShell.Main > Box で maw={1440} mx="auto" px="md" */}
      <Grid gutter="md" align="stretch" style={{ flex: 1, minHeight: 0 }}>
        {/* 左カラム (span=5 で 5:7 の比率)*/}
        <Grid.Col span={5} style={{ display: "flex", flexDirection: "column", minHeight: 0 }}>
          <Stack gap="md" h="100%" style={{ flex: 1, minHeight: 0 }}>
            <Box
              style={{
                height: `${ModelHeight}`, // ヘッダー高さを引いた残りの高さ
                display: "flex",
                flexDirection: "column",
              }}
            >
              {/* 上段（Live2Dなど）を伸ばす*/}
              <Card
                withBorder
                radius="md"
                className="layout-frame model-frame"
                style={{ flex: 1, overflow: "hidden" }} // 重要: flex:1 で高さを自動調整
              >
                <Box>
                  {modelComponent || (
                    <Stack align="center" justify="center">
                      <Text size="lg" c="white">
                        Model Display Area
                      </Text>
                      <Text size="sm" c="white" opacity={0.8}>
                        Live2D model will be displayed here
                      </Text>
                    </Stack>
                  )}
                </Box>
              </Card>
            </Box>

            {/* 下段 2 カード  */}
            <Group grow align="stretch" style={{ height: ScheduleHeight }}>
              <Card
                withBorder
                radius="md"
                className="layout-frame schedule-frame"
                h="100%"
                style={{ display: "flex", flexDirection: "column", overflow: "hidden" }}
              >
                <Title order={5} className="frame-title">
                  Schedule
                </Title>
                <ScrollArea style={{ flex: 1, minHeight: 0 }} offsetScrollbars>
                  {scheduleComponent || (
                    <Stack gap="xs" mt="sm">
                      <Card p="xs" style={{ background: "rgba(255, 255, 255, 0.2)" }}>
                        <Text size="sm" c="white">
                          10:00
                        </Text>
                      </Card>
                      <Card p="xs" style={{ background: "rgba(255, 255, 255, 0.2)" }}>
                        <Text size="sm" c="white">
                          11:00
                        </Text>
                      </Card>
                      <Card p="xs" style={{ background: "rgba(255, 255, 255, 0.2)" }}>
                        <Text size="sm" c="white">
                          12:00
                        </Text>
                      </Card>
                      <Card p="xs" style={{ background: "rgba(255, 255, 255, 0.2)" }}>
                        <Text size="sm" c="white">
                          12:00
                        </Text>
                      </Card>
                      <Card p="xs" style={{ background: "rgba(255, 255, 255, 0.2)" }}>
                        <Text size="sm" c="white">
                          12:00
                        </Text>
                      </Card>
                      <Card p="xs" style={{ background: "rgba(255, 255, 255, 0.2)" }}>
                        <Text size="sm" c="white">
                          12:00
                        </Text>
                      </Card>
                    </Stack>
                  )}
                </ScrollArea>
              </Card>

              <Card
                withBorder
                radius="md"
                className="layout-frame text-frame"
                h="100%"
                style={{ display: "flex", flexDirection: "column" }}
              >
                <Title order={5} className="frame-title">
                  Speech to Text / Translation
                </Title>
                <Box
                  style={{
                    flex: 1,
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                  }}
                >
                  {textComponent || (
                    <Stack align="center" justify="center">
                      <Text size="md" fw={500} c="white">
                        Hello, how are you?
                      </Text>
                      <Text size="sm" c="white" opacity={0.8}>
                        こんにちは、お元気ですか？
                      </Text>
                    </Stack>
                  )}
                </Box>
              </Card>
            </Group>
          </Stack>
        </Grid.Col>

        {/* 右カラム (span=7) - チャット*/}
        <Grid.Col
          span={7}
          style={{
            display: "flex",
            flexDirection: "column",
            minHeight: 0,
          }}
        >
          <Box
            style={{
              height: `calc(${ScheduleHeight} + ${ModelHeight} + 16px)`, // ヘッダー高さを引いた残りの高さ
              display: "flex",
              flexDirection: "column",
            }}
          >
            <Card
              withBorder
              radius="md"
              className="layout-frame chat-frame"
              style={{
                display: "flex", // 重要: 3段構成（Header/Messages/Input）用
                flexDirection: "column",
                overflow: "hidden", // 重要: 内容の溢れを防ぐ
                flex: 1, // 重要: カラム高さいっぱいに伸ばす
                minHeight: 0, // 重要: flexレイアウトの計算を正しく行う
              }}
            >
              <Title order={5} className="frame-title">
                Chat
              </Title>

              {/* チャット内容エリア */}
              <Box style={{ flex: 1, minHeight: 0, display: "flex", flexDirection: "column" }}>
                {chatComponent}
              </Box>
            </Card>
          </Box>
        </Grid.Col>
      </Grid>
    </Box>
  );
};
