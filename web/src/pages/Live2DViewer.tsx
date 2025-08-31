import React, { useState, useRef } from "react";
import { Box, Grid } from "@mantine/core";
import { Live2DModelViewer } from "../components/Live2DModelViewer";
import { Live2DControls } from "../components/Live2DControls";
import type { ModelParameter, ModelMotion, ModelExpression } from "../components/Live2DModelViewer";
import type { Live2DModel } from "pixi-live2d-display-lipsyncpatch/cubism4";

export const Live2DViewer: React.FC = () => {
  const [speechText] = useState("こんにちは！Live2D Viewerです。");
  const [modelInfo, setModelInfo] = useState<{
    parameters: ModelParameter[];
    motions: ModelMotion[];
    expressions: ModelExpression[];
  }>({
    parameters: [],
    motions: [],
    expressions: [],
  });
  const modelRef = useRef<Live2DModel | null>(null);

  const handleModelInfoUpdate = (info: {
    parameters: ModelParameter[];
    motions: ModelMotion[];
    expressions: ModelExpression[];
  }) => {
    setModelInfo(info);
    console.log("Model info updated:", {
      parameters: info.parameters.length,
      motions: info.motions.length,
      expressions: info.expressions.length,
    });
  };

  return (
    <Box
      style={{
        width: "100%",
        height: "1024",
        backgroundColor: "#1a1b1e",
        overflow: "hidden",
      }}
    >
      <Box
        style={{
          flex: 1,
          minHeight: 0,
          padding: "20px",
          overflow: "hidden",
        }}
      >
        <Grid style={{ height: "100%" }} gutter="md">
          <Grid.Col span={8}>
            <Box
              style={{
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
              }}
            >
              <Live2DModelViewer
                width={700}
                height={500}
                speechText={speechText}
                isSpeaking={true}
                bubbleSide="bottom"
                useCard={true}
                cardTitle="ASSISTANT"
                onModelInfoUpdate={handleModelInfoUpdate}
                modelRef={modelRef}
              />
            </Box>
          </Grid.Col>
          <Grid.Col span={4}>
            <Box
              style={{
                minHeight: 0,
                overflow: "hidden",
                display: "flex",
                flexDirection: "column",
              }}
            >
              <Live2DControls
                parameters={modelInfo.parameters}
                motions={modelInfo.motions}
                expressions={modelInfo.expressions}
                modelRef={modelRef}
              />
            </Box>
          </Grid.Col>
        </Grid>
      </Box>
    </Box>
  );
};
