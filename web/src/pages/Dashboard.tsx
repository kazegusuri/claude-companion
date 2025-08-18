import React, { useState, useEffect } from "react";
import { MainLayout } from "../components/Layout/MainLayout";
import { Live2DModelViewer } from "../components/Live2DModelViewer";
import { ChatDisplay } from "../components/ChatDisplay";
import { ActionIcon, Stack, Tooltip } from "@mantine/core";
import { IconMessage, IconMessageDown, IconMessageOff } from "@tabler/icons-react";

type BubbleState = "right" | "bottom" | "hidden";

export const Dashboard: React.FC = () => {
  const [speechText, setSpeechText] = useState(
    "修正と確認を行った結果、レイアウトが正しく修正されました。モデルや吹き出し、カードが適切な位置とサイズで表示されています。これからも改善を続けます。",
  );
  const [bubbleState, setBubbleState] = useState<BubbleState>("bottom"); // 初期状態で下側表示

  // 3段階トグル: 右側 → 下側 → 非表示 → 右側...
  const toggleBubble = () => {
    setBubbleState((prev) => {
      switch (prev) {
        case "right":
          console.log("吹き出しを下側に表示");
          return "bottom";
        case "bottom":
          console.log("吹き出しを非表示");
          return "hidden";
        case "hidden":
          console.log("吹き出しを右側に表示");
          return "right";
      }
    });
  };

  // アイコンとツールチップのテキストを決定
  const getIconAndTooltip = () => {
    switch (bubbleState) {
      case "right":
        return {
          icon: <IconMessage size={18} />,
          tooltip: "吹き出し：右側表示中 → クリックで下側へ",
        };
      case "bottom":
        return {
          icon: <IconMessageDown size={18} />,
          tooltip: "吹き出し：下側表示中 → クリックで非表示",
        };
      case "hidden":
        return {
          icon: <IconMessageOff size={18} />,
          tooltip: "吹き出し：非表示 → クリックで右側へ",
        };
    }
  };

  const { icon, tooltip } = getIconAndTooltip();

  return (
    <MainLayout
      style={{ flex: 1, height: "100%", overflow: "hidden" }}
      modelComponent={
        <div
          style={{
            flex: 1,
            minHeight: 0,
            display: "flex",
            flexDirection: "row",
            justifyContent: "space-between",
            padding: "10px",
          }}
        >
          <div
            style={{
              flex: 1,
              height: "100%",
              minHeight: 0,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <Live2DModelViewer
              speechText={speechText}
              isSpeaking={bubbleState !== "hidden"}
              bubbleSide={bubbleState === "hidden" ? "bottom" : bubbleState}
              useCard={true}
              cardTitle="ASSISTANT"
            />
          </div>
          <div
            style={{
              minHeight: 0,
              display: "flex",
              flexDirection: "column",
              alignItems: "flex-end",
              justifyContent: "flex-end",
              paddingBottom: "5px",
            }}
          >
            <Tooltip label={tooltip} position="top" withArrow>
              <ActionIcon
                onClick={toggleBubble}
                size="sm"
                radius="xl"
                variant="filled"
                style={{
                  zIndex: 1001,
                }}
              >
                {icon}
              </ActionIcon>
            </Tooltip>
          </div>
        </div>
      }
      scheduleComponent={null}
      textComponent={null}
      chatComponent={<ChatDisplay />}
    />
  );
};
