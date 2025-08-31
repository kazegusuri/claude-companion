import { useState } from "react";
import { MantineProvider, createTheme, AppShell, Box } from "@mantine/core";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import "@mantine/core/styles.css";
import { AudioNarrator } from "@components/AudioNarrator";
import { Dashboard } from "./pages/Dashboard";
import { MobileDashboard } from "./pages/MobileDashboard";
import { Live2DViewer } from "./pages/Live2DViewer";
import { AppHeader } from "./components/AppHeader";
import "./App.css";

const theme = createTheme({});

function DesktopApp() {
  const [currentView, setCurrentView] = useState<"dashboard" | "narrator" | "live2d">("dashboard");

  return (
    <AppShell header={{ height: 60 }} padding={0} style={{ height: "100dvh", overflow: "hidden" }}>
      <AppShell.Header p={0}>
        <AppHeader currentView={currentView} onViewChange={setCurrentView} />
      </AppShell.Header>

      <AppShell.Main style={{ display: "flex", minHeight: 0, overflow: "hidden" }}>
        <Box
          maw={1440}
          w="100%"
          mx="auto"
          px="md"
          style={{ flex: 1, minHeight: 0, overflow: "hidden" }}
        >
          {currentView === "dashboard" && <Dashboard />}
          {currentView === "narrator" && <AudioNarrator />}
          {currentView === "live2d" && <Live2DViewer />}
        </Box>
      </AppShell.Main>
    </AppShell>
  );
}

function App() {
  return (
    <MantineProvider theme={theme} defaultColorScheme="dark">
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<DesktopApp />} />
          <Route path="/mobile" element={<MobileDashboard />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </MantineProvider>
  );
}

export default App;
