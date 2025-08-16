import { useState } from "react";
import { MantineProvider, createTheme, AppShell } from "@mantine/core";
import "@mantine/core/styles.css";
import { Live2DViewer } from "@components/Live2DViewer";
import { AudioNarrator } from "@components/AudioNarrator";
import { Dashboard } from "./pages/Dashboard";
import { AppHeader } from "./components/AppHeader";
import "./App.css";

const theme = createTheme({});

function App() {
  const [currentView, setCurrentView] = useState<"dashboard" | "narrator" | "live2d">("dashboard");

  return (
    <MantineProvider theme={theme} defaultColorScheme="dark">
      <AppShell
        header={{ height: 60 }}
        padding={0}
        styles={{
          main: {
            backgroundColor: '#0a0a0a',
          },
        }}
      >
        <AppShell.Header>
          <AppHeader currentView={currentView} onViewChange={setCurrentView} />
        </AppShell.Header>

        <AppShell.Main>
          {currentView === "dashboard" && <Dashboard />}
          {currentView === "narrator" && <AudioNarrator />}
          {currentView === "live2d" && <Live2DViewer />}
        </AppShell.Main>
      </AppShell>
    </MantineProvider>
  );
}

export default App;
