import React from "react";
import { MainLayout } from "../components/Layout/MainLayout";
import { Live2DModelViewer } from "../components/Live2DModelViewer";
import { ChatDisplay } from "../components/ChatDisplay";

export const Dashboard: React.FC = () => {
  return (
    <MainLayout
      style={{ flex: 1, minHeight: 0, overflow: "hidden" }}
      modelComponent={<Live2DModelViewer />}
      scheduleComponent={null}
      textComponent={null}
      chatComponent={<ChatDisplay />}
    />
  );
};
