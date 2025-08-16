import { useState } from "react";
import { Live2DViewer } from "@components/Live2DViewer";
import { AudioNarrator } from "@components/AudioNarrator";
import "./App.css";

function App() {
  const [showAudioNarrator, setShowAudioNarrator] = useState(true);

  return (
    <div className="app">
      <h1>Claude Companion Web</h1>

      <div className="tab-buttons">
        <button
          className={showAudioNarrator ? "active" : ""}
          onClick={() => setShowAudioNarrator(true)}
        >
          Audio Narrator
        </button>
        <button
          className={!showAudioNarrator ? "active" : ""}
          onClick={() => setShowAudioNarrator(false)}
        >
          Live2D Viewer
        </button>
      </div>

      <div className="content">{showAudioNarrator ? <AudioNarrator /> : <Live2DViewer />}</div>
    </div>
  );
}

export default App;
