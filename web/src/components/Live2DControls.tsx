import { Badge, Button, Card, Group, ScrollArea, Stack, Table, Tabs, Text } from "@mantine/core";
import { IconMoodSmile, IconPlayerPlay } from "@tabler/icons-react";
import type { Live2DModel } from "pixi-live2d-display-lipsyncpatch/cubism4";
import type React from "react";
import { useState } from "react";
import type { ModelExpression, ModelMotion, ModelParameter } from "./Live2DModelViewer";

interface Live2DControlsProps {
  parameters: ModelParameter[];
  motions: ModelMotion[];
  expressions: ModelExpression[];
  modelRef: React.MutableRefObject<Live2DModel | null>;
}

export const Live2DControls: React.FC<Live2DControlsProps> = ({
  parameters,
  motions,
  expressions,
  modelRef,
}) => {
  const [activeTab, setActiveTab] = useState<string | null>("parameters");

  const handleMotionPlay = (group: string, index: number) => {
    if (modelRef.current) {
      try {
        modelRef.current.motion(group, index);
        console.log(`Playing motion: ${group}[${index}]`);
      } catch (error) {
        console.error(`Failed to play motion ${group}[${index}]:`, error);
      }
    }
  };

  const handleExpressionSet = (name: string) => {
    if (modelRef.current) {
      try {
        modelRef.current.expression(name);
        console.log(`Setting expression: ${name}`);
      } catch (error) {
        console.error(`Failed to set expression ${name}:`, error);
      }
    }
  };

  // モーションをグループごとに整理
  const motionGroups = motions.reduce(
    (acc, motion) => {
      if (!acc[motion.group]) {
        acc[motion.group] = [];
      }
      acc[motion.group]?.push(motion);
      return acc;
    },
    {} as Record<string, typeof motions>,
  );

  return (
    <Card
      withBorder
      style={{
        minHeight: 0,
        height: 800,
        display: "flex",
        flexDirection: "column",
        overflow: "hidden",
      }}
    >
      <Tabs
        value={activeTab}
        onChange={setActiveTab}
        style={{
          minHeight: 0,
          display: "flex",
          flexDirection: "column",
          overflow: "hidden",
        }}
      >
        <Tabs.List style={{ flex: "0 0 auto" }}>
          <Tabs.Tab
            value="parameters"
            rightSection={
              <Badge size="xs" variant="filled">
                {parameters.length}
              </Badge>
            }
          >
            パラメータ
          </Tabs.Tab>
          <Tabs.Tab
            value="motions"
            rightSection={
              <Badge size="xs" variant="filled">
                {motions.length}
              </Badge>
            }
          >
            モーション
          </Tabs.Tab>
          <Tabs.Tab
            value="expressions"
            rightSection={
              <Badge size="xs" variant="filled">
                {expressions.length}
              </Badge>
            }
          >
            表情
          </Tabs.Tab>
        </Tabs.List>

        {/* パラメータタブ */}
        <Tabs.Panel
          value="parameters"
          style={{
            flex: 1,
            minHeight: 0,
            paddingTop: "12px",
            display: "flex",
            flexDirection: "column",
            overflow: "hidden",
          }}
        >
          <ScrollArea
            style={{
              flex: 1,
              minHeight: 0,
            }}
            offsetScrollbars
            scrollbarSize={8}
          >
            <Table striped highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>パラメータ名</Table.Th>
                  <Table.Th>値</Table.Th>
                  <Table.Th>範囲</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {parameters.map((param) => (
                  <Table.Tr key={param.id}>
                    <Table.Td style={{ fontSize: "12px", wordBreak: "break-all" }}>
                      {param.name}
                    </Table.Td>
                    <Table.Td>
                      <Badge size="xs" variant="light">
                        {param.value.toFixed(2)}
                      </Badge>
                    </Table.Td>
                    <Table.Td>
                      <Text size="xs" c="dimmed">
                        {param.min.toFixed(1)} ~ {param.max.toFixed(1)}
                      </Text>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          </ScrollArea>
        </Tabs.Panel>

        {/* モーションタブ */}
        <Tabs.Panel
          value="motions"
          style={{
            flex: 1,
            minHeight: 0,
            paddingTop: "12px",
            display: "flex",
            flexDirection: "column",
            overflow: "hidden",
          }}
        >
          <ScrollArea
            style={{
              flex: 1,
              minHeight: 0,
            }}
            offsetScrollbars
            scrollbarSize={8}
          >
            <Stack gap="md">
              {Object.entries(motionGroups).map(([group, groupMotions]) => (
                <div key={group}>
                  <Group gap="xs" mb="xs">
                    <Text size="sm" fw={600}>
                      {group}
                    </Text>
                    <Badge size="xs" variant="dot">
                      {groupMotions.length}
                    </Badge>
                  </Group>
                  <Stack gap="xs">
                    {groupMotions.map((motion) => (
                      <Button
                        key={`${motion.group}_${motion.index}`}
                        size="xs"
                        variant="light"
                        fullWidth
                        leftSection={<IconPlayerPlay size={14} />}
                        onClick={() => handleMotionPlay(motion.group, motion.index)}
                        styles={{
                          label: { justifyContent: "flex-start" },
                        }}
                      >
                        {motion.name}
                      </Button>
                    ))}
                  </Stack>
                </div>
              ))}
              {motions.length === 0 && (
                <Text size="sm" c="dimmed" ta="center" mt="md">
                  モーションデータが見つかりません
                </Text>
              )}
            </Stack>
          </ScrollArea>
        </Tabs.Panel>

        {/* エクスプレッションタブ */}
        <Tabs.Panel
          value="expressions"
          style={{
            flex: 1,
            minHeight: 0,
            paddingTop: "12px",
            display: "flex",
            flexDirection: "column",
            overflow: "hidden",
          }}
        >
          <ScrollArea
            style={{
              flex: 1,
              minHeight: 0,
            }}
            offsetScrollbars
            scrollbarSize={8}
          >
            <Stack gap="xs">
              {expressions.map((exp) => (
                <Button
                  key={exp.name}
                  size="sm"
                  variant="light"
                  fullWidth
                  leftSection={<IconMoodSmile size={16} />}
                  onClick={() => handleExpressionSet(exp.name)}
                  styles={{
                    label: { justifyContent: "flex-start" },
                  }}
                >
                  {exp.name}
                </Button>
              ))}
              {expressions.length === 0 && (
                <Text size="sm" c="dimmed" ta="center" mt="md">
                  表情データが見つかりません
                </Text>
              )}
            </Stack>
          </ScrollArea>
        </Tabs.Panel>
      </Tabs>
    </Card>
  );
};
