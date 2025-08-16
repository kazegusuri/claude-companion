import React from 'react';
import { Grid, Paper, Title, Text, Stack, Box, ScrollArea } from '@mantine/core';
import './MainLayout.css';

interface MainLayoutProps {
  modelComponent?: React.ReactNode;
  scheduleComponent?: React.ReactNode;
  textComponent?: React.ReactNode;
  chatComponent?: React.ReactNode;
}

export const MainLayout: React.FC<MainLayoutProps> = ({
  modelComponent,
  scheduleComponent,
  textComponent,
  chatComponent,
}) => {
  return (
    <Box className="main-layout" style={{ display: 'flex', flexDirection: 'column', overflow: 'hidden', height: '100%' }}>
      <Grid gutter="md" justify="center" style={{ flex: 1, minHeight: 0, height: '100%', maxWidth: '1600px', margin: '0 auto', alignItems: 'flex-start' }} p="xs">
        {/* Left Area - Model, Schedule, and Text */}
        <Grid.Col span="auto" style={{ width: '600px', display: 'flex', flexShrink: 0 }}>
          <Stack gap="xs" style={{ width: '100%' }}>
            {/* Model Display Area - Top */}
            <Paper 
              className="layout-frame model-frame" 
              withBorder 
              p="md" 
              h={550}
            >
              <Box className="frame-content" style={{ height: '100%' }}>
                {modelComponent || (
                  <Stack align="center" justify="center" h="100%">
                    <Text size="lg" c="white">Model Display Area</Text>
                    <Text size="sm" c="white" opacity={0.8}>Live2D model will be displayed here</Text>
                  </Stack>
                )}
              </Box>
            </Paper>

            {/* Bottom Area - Schedule and Text side by side */}
            <Grid gutter="xs" style={{ height: '220px' }}>
              {/* Schedule Display Area - Bottom Left */}
              <Grid.Col span={6}>
                <Paper 
                  className="layout-frame schedule-frame" 
                  withBorder 
                  p="sm" 
                  style={{ height: '100%', overflow: 'hidden', boxSizing: 'border-box' }}
                >
                  <Title order={5} className="frame-title">Schedule</Title>
                  <ScrollArea h="calc(100% - 30px)" offsetScrollbars>
                    {scheduleComponent || (
                      <Stack gap="xs" mt="sm">
                        <Paper p="xs" style={{ background: 'rgba(255, 255, 255, 0.2)' }}>
                          <Text size="sm" c="white">10:00</Text>
                        </Paper>
                        <Paper p="xs" style={{ background: 'rgba(255, 255, 255, 0.2)' }}>
                          <Text size="sm" c="white">11:00</Text>
                        </Paper>
                        <Paper p="xs" style={{ background: 'rgba(255, 255, 255, 0.2)' }}>
                          <Text size="sm" c="white">12:00</Text>
                        </Paper>
                      </Stack>
                    )}
                  </ScrollArea>
                </Paper>
              </Grid.Col>

              {/* Text/Translation Display Area - Bottom Right */}
              <Grid.Col span={6}>
                <Paper className="layout-frame text-frame" withBorder p="sm" style={{ height: '100%', overflow: 'hidden', boxSizing: 'border-box' }}>
                  <Title order={5} className="frame-title">Speech to Text / Translation</Title>
                  <Box className="frame-content">
                    {textComponent || (
                      <Stack align="center" justify="center" h="100%">
                        <Text size="md" fw={500} c="white">Hello, how are you?</Text>
                        <Text size="sm" c="white" opacity={0.8}>こんにちは、お元気ですか？</Text>
                      </Stack>
                    )}
                  </Box>
                </Paper>
              </Grid.Col>
            </Grid>
          </Stack>
        </Grid.Col>

        {/* Right Column - Chat */}
        <Grid.Col span="auto" style={{ width: '900px', height: '100%', display: 'flex', flexDirection: 'column', flexShrink: 0 }}>
          <Paper className="layout-frame chat-frame" withBorder style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
            <Title order={5} className="frame-title">Chat</Title>
            {chatComponent || (
              <Stack gap="xs" mt="sm" style={{ flex: 1 }}>
                <Paper p="sm" style={{ background: 'rgba(255, 255, 255, 0.2)' }}>
                  <Text size="sm" c="white">Lorem ipsum dolor sit amet, consectetur adipiscing elit.</Text>
                </Paper>
                <Paper p="sm" style={{ background: 'rgba(255, 255, 255, 0.2)' }}>
                  <Text size="sm" c="white">Lorem ipsum dolor sit amet, consectetur adipiscing elit.</Text>
                </Paper>
                <Paper p="sm" style={{ background: 'rgba(255, 255, 255, 0.2)' }}>
                  <Text size="sm" c="white">Lorem ipsum dolor sit amet, consectetur adipiscing elit.</Text>
                </Paper>
                <Paper p="sm" style={{ background: 'rgba(255, 255, 255, 0.2)' }}>
                  <Text size="sm" c="white">Lorem ipsum dolor sit amet, consectetur adipiscing elit.</Text>
                </Paper>
              </Stack>
            )}
          </Paper>
        </Grid.Col>
      </Grid>
    </Box>
  );
};