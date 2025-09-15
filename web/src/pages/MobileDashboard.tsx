import { Badge, Box, Group, Paper, Text } from "@mantine/core";
import { IconRobot, IconTerminal } from "@tabler/icons-react";
import type React from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { ChatDisplay } from "../components/ChatDisplay";
import { Live2DModelViewer } from "../components/Live2DModelViewer";
import type { Agent } from "../services/AgentService";
import { AgentService } from "../services/AgentService";
import { messageRouter } from "../services/MessageRouter";
import type { ChatMessage, ConnectionStatus } from "../services/WebSocketClient";
import { WebSocketAudioClient } from "../services/WebSocketClient";
import styles from "./MobileDashboard.module.css";

// パルスアニメーション用のスタイル
const pulseKeyframes = `
  @keyframes pulse {
    0% {
      box-shadow: 0 0 0 0 rgba(255, 193, 7, 0.4);
    }
    70% {
      box-shadow: 0 0 0 10px rgba(255, 193, 7, 0);
    }
    100% {
      box-shadow: 0 0 0 0 rgba(255, 193, 7, 0);
    }
  }
`;

// モバイル用のコンパクトなAgentListコンポーネント
interface MobileAgentListProps {
  onAgentClick?: (agent: Agent) => void;
  selectedAgentPID?: number | null;
  wsClient?: WebSocketAudioClient | null;
  maxAgents?: number;
}

const MobileAgentList: React.FC<MobileAgentListProps> = ({
  onAgentClick,
  selectedAgentPID,
  wsClient,
  maxAgents = 2,
}) => {
  const [agents, setAgents] = useState<Agent[]>([]);
  const agentService = useMemo(() => new AgentService(), []);

  // エージェント一覧を取得
  const fetchAgents = useCallback(async () => {
    try {
      const data = await agentService.getAgents();
      // 最大表示数に制限
      const limitedAgents = data.slice(0, maxAgents);
      setAgents(limitedAgents);
    } catch (error) {
      console.error("Failed to fetch agents:", error);
    }
  }, [agentService, maxAgents]);

  // 初回取得と定期更新
  useEffect(() => {
    fetchAgents();

    // 3秒ごとに更新（モバイルは更新頻度を少し高めに）
    const interval = setInterval(fetchAgents, 3000);

    return () => clearInterval(interval);
  }, [fetchAgents]);

  // WebSocketメッセージを監視してエージェントリストを即座に更新
  useEffect(() => {
    if (!wsClient) return;

    const handleMessage = (message: ChatMessage) => {
      // エージェント関連のイベントで即座に更新
      if (
        message.metadata?.eventType === "tool_permission" ||
        message.metadata?.agentPid ||
        message.type === "notification"
      ) {
        fetchAgents();
      }
    };

    const cleanup = wsClient.addMessageListener(handleMessage);
    return cleanup;
  }, [wsClient, fetchAgents]);

  return (
    <>
      <style>{pulseKeyframes}</style>
      <Box style={{ width: "100%" }}>
        {/* ヘッダー */}
        <Group gap="xs" mb={4}>
          <Text size="sm" fw={600} c="white">
            Agents
          </Text>
        </Group>

        {/* エージェントリスト（1行1エージェント、最大2行） */}
        <Box style={{ display: "flex", flexDirection: "column", gap: "4px" }}>
          {agents.map((agent) => (
            <Paper
              key={agent.pid}
              p="4px 8px"
              style={{
                backgroundColor: agent.session?.activeTool?.isWaitingApproval
                  ? "rgba(255, 193, 7, 0.15)" // 黄色の背景（許可待ち）
                  : selectedAgentPID === agent.pid
                    ? "rgba(139, 92, 246, 0.2)"
                    : "rgba(255, 255, 255, 0.05)",
                border: agent.session?.activeTool?.isWaitingApproval
                  ? "1px solid rgba(255, 193, 7, 0.5)" // 黄色の枠線（許可待ち）
                  : selectedAgentPID === agent.pid
                    ? "1px solid rgba(139, 92, 246, 0.5)"
                    : "1px solid rgba(255, 255, 255, 0.1)",
                cursor: "pointer",
                transition: "all 0.2s",
                display: "flex",
                alignItems: "center",
                gap: "8px",
                width: "100%",
                animation: agent.session?.activeTool?.isWaitingApproval
                  ? "pulse 2s infinite"
                  : "none",
              }}
              onClick={() => onAgentClick?.(agent)}
            >
              <IconTerminal size={16} style={{ flexShrink: 0 }} />
              <Box
                style={{ flex: 1, minWidth: 0, display: "flex", alignItems: "center", gap: "8px" }}
              >
                <Text size="xs" truncate style={{ flex: 1 }}>
                  {agent.projectName || "claude-companion"}
                </Text>
                <Group gap={4} style={{ flexShrink: 0 }}>
                  <Badge size="xs" color="gray" variant="filled">
                    PID: {agent.pid}
                  </Badge>
                  {agent.session?.activeTool?.isWaitingApproval && (
                    <Badge size="xs" color="yellow" variant="filled">
                      🔐 許可待ち
                    </Badge>
                  )}
                  {agent.session?.activeTool && !agent.session?.activeTool?.isWaitingApproval && (
                    <Badge size="xs" color="blue" variant="light">
                      {agent.session.activeTool.toolName}
                    </Badge>
                  )}
                </Group>
              </Box>
            </Paper>
          ))}
          {agents.length === 0 && (
            <Paper
              p="4px 8px"
              style={{
                backgroundColor: "rgba(255, 255, 255, 0.05)",
                border: "1px solid rgba(255, 255, 255, 0.1)",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                width: "100%",
              }}
            >
              <Text size="xs" c="dimmed">
                No agents running
              </Text>
            </Paper>
          )}
        </Box>
      </Box>
    </>
  );
};

export const MobileDashboard: React.FC = () => {
  const [searchParams] = useSearchParams();
  const [speechText, setSpeechText] = useState("音声を待機中...");
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>("disconnected");
  const [currentMessageId, setCurrentMessageId] = useState<string | null>(null);
  const [windowSize, setWindowSize] = useState({ width: 0, height: 0 });
  const [isAudioEnabled, setIsAudioEnabled] = useState(false); // 初期状態では無効（ユーザーインタラクションが必要）
  const [currentAudioData, setCurrentAudioData] = useState<string | undefined>(undefined);
  const [selectedAgent, setSelectedAgent] = useState<Agent | null>(null); // Track selected agent
  const [isPlayingAudio, setIsPlayingAudio] = useState(false); // 音声再生状態
  const [showSpeechBubble, setShowSpeechBubble] = useState(false); // 吹き出し表示状態
  const speechBubbleTimerRef = useRef<NodeJS.Timeout | null>(null); // 吹き出し非表示タイマー

  // URLパラメータから指定された幅と高さを取得（デフォルトは400x1280）
  const specifiedDimensions = useMemo(() => {
    const width = parseInt(searchParams.get("width") || "400", 10);
    const height = parseInt(searchParams.get("height") || "1280", 10);
    return { width, height };
  }, [searchParams]);

  const wsClient = useRef<WebSocketAudioClient | null>(null);
  const audioQueue = useRef<ChatMessage[]>([]);
  const isProcessingQueue = useRef(false);

  // processAudioQueueのrefを作成
  const processAudioQueueRef = useRef<(() => void) | null>(null);

  // 音声が無効になったらキューをクリア
  useEffect(() => {
    if (!isAudioEnabled) {
      audioQueue.current = [];
      setCurrentMessageId(null);
      setCurrentAudioData(undefined);
    }
  }, [isAudioEnabled]);

  // isAudioEnabledをrefで管理して、関数の再作成を防ぐ
  const isAudioEnabledRef = useRef(isAudioEnabled);
  useEffect(() => {
    isAudioEnabledRef.current = isAudioEnabled;
  }, [isAudioEnabled]);

  // 音声キューを処理（一度だけ作成される関数）
  const processAudioQueue = useCallback(() => {
    if (isProcessingQueue.current || audioQueue.current.length === 0) {
      return;
    }

    if (!isAudioEnabledRef.current) {
      audioQueue.current = [];
      setCurrentAudioData(undefined);
      return;
    }

    const message = audioQueue.current[0];

    if (message?.audioData) {
      // メッセージの情報を保持してから削除
      const messageId = message.id;
      const audioData = message.audioData;

      // キューから削除
      audioQueue.current.shift();
      isProcessingQueue.current = true;

      // メッセージIDと音声データを設定
      setCurrentMessageId(messageId || null);
      setCurrentAudioData(audioData);

      // 音声再生開始時に吹き出しを表示
      setIsPlayingAudio(true);
      setShowSpeechBubble(true);

      // タイマーがあればクリア
      if (speechBubbleTimerRef.current) {
        clearTimeout(speechBubbleTimerRef.current);
        speechBubbleTimerRef.current = null;
      }
    } else {
      audioQueue.current.shift();
      // 次のメッセージを処理
      if (audioQueue.current.length > 0) {
        setTimeout(() => processAudioQueueRef.current?.(), 0);
      }
    }
  }, []); // 空の依存配列で一度だけ作成

  // processAudioQueueをrefに保存
  useEffect(() => {
    processAudioQueueRef.current = processAudioQueue;
  }, [processAudioQueue]);

  // 音声再生終了時の処理
  const handleAudioEnd = useCallback(() => {
    // 既にキューから削除済みなので、削除処理は不要
    setCurrentMessageId(null);
    setCurrentAudioData(undefined);
    isProcessingQueue.current = false;
    setIsPlayingAudio(false);

    // 2秒後に吹き出しを非表示にする
    if (speechBubbleTimerRef.current) {
      clearTimeout(speechBubbleTimerRef.current);
    }
    speechBubbleTimerRef.current = setTimeout(() => {
      setShowSpeechBubble(false);
      speechBubbleTimerRef.current = null;
    }, 2000);

    // 次のアイテムを処理
    if (audioQueue.current.length > 0) {
      setTimeout(() => processAudioQueueRef.current?.(), 100);
    }
  }, []);

  // WebSocketメッセージハンドラー（依存配列を空にして再作成を防ぐ）
  const handleWebSocketMessage = useCallback((message: ChatMessage) => {
    // テキストを更新（assistantメッセージのみ）
    if (message.text && message.role === "assistant") {
      setSpeechText(message.text);
    }

    // 音声データがある場合はキューに追加
    // Check for assistant messages with audio subtype or legacy audio type
    if (
      (message.type === "audio" || (message.type === "assistant" && message.subType === "audio")) &&
      message.audioData &&
      isAudioEnabledRef.current // refを使用
    ) {
      // 既存のメッセージがキューにないか確認
      const isDuplicate = audioQueue.current.some((msg) => msg.id === message.id);

      if (!isDuplicate) {
        audioQueue.current.push(message);

        // 優先度でソート（現在は全て同じ優先度なので実質的に追加順）
        audioQueue.current.sort((a, b) => b.priority - a.priority);

        // 処理中でない場合のみprocessAudioQueueを呼ぶ
        if (!isProcessingQueue.current) {
          processAudioQueueRef.current?.();
        }
      }
    }
  }, []); // 空の依存配列で一度だけ作成

  // 音声出力のトグル
  const handleAudioToggle = useCallback(() => {
    setIsAudioEnabled((prev) => {
      const newState = !prev;

      // 音声を無効にした場合、再生中の音声を停止し、キューをクリア
      if (!newState) {
        audioQueue.current = [];
        setCurrentMessageId(null);
        setCurrentAudioData(undefined);
        isProcessingQueue.current = false;
      }

      return newState;
    });
  }, []);

  // メッセージハンドラーのrefを作成
  const messageHandlerRef = useRef<(message: ChatMessage) => void>();

  // メッセージハンドラーをrefに保存
  useEffect(() => {
    messageHandlerRef.current = handleWebSocketMessage;
  }, [handleWebSocketMessage]);

  // WebSocket接続の初期化（単一接続を維持）
  useEffect(() => {
    let mounted = true;
    let client: WebSocketAudioClient | null = null;

    // 少し遅延させてStrictModeの二重レンダリングの影響を軽減
    const timer = setTimeout(() => {
      if (!mounted) return;

      // WebSocketクライアントを作成
      const wsUrl = import.meta.env.VITE_WS_URL || "ws://localhost:8080/ws/audio";
      client = new WebSocketAudioClient(
        wsUrl,
        (message) => messageHandlerRef.current?.(message),
        setConnectionStatus,
      );

      // WebSocketに接続
      client.connect();

      // stateにセット
      wsClient.current = client;
    }, 0);

    // クリーンアップ
    return () => {
      mounted = false;
      clearTimeout(timer);
      if (client) {
        client.disconnect();
        wsClient.current = null;
      }
    };
  }, []); // 空の依存配列で一度だけ実行

  // ビューポートスケーリング計算とサイズ追跡
  useEffect(() => {
    const updateScale = () => {
      const viewportWidth = window.innerWidth;
      const viewportHeight = window.innerHeight;

      // サイズ情報を更新
      setWindowSize({ width: viewportWidth, height: viewportHeight });

      if (viewportWidth < 400) {
        const scale = viewportWidth / 400;
        document.documentElement.style.setProperty("--scale-factor", scale.toString());
      } else {
        document.documentElement.style.setProperty("--scale-factor", "1");
      }
    };

    updateScale();
    window.addEventListener("resize", updateScale);
    return () => window.removeEventListener("resize", updateScale);
  }, []);

  // コンポーネントのアンマウント時にタイマーをクリア
  useEffect(() => {
    return () => {
      if (speechBubbleTimerRef.current) {
        clearTimeout(speechBubbleTimerRef.current);
        speechBubbleTimerRef.current = null;
      }
    };
  }, []);

  return (
    <Box
      className={styles.mobileContainer || ""}
      style={{
        backgroundColor: "#1a1b1e",
        display: "flex",
        flexDirection: "column",
        position: "relative",
      }}
    >
      {/* デバッグ情報: 画面サイズ表示 */}
      <Box
        style={{
          position: "absolute",
          top: 0,
          left: 0,
          right: 0,
          backgroundColor: "rgba(0, 0, 0, 0.7)",
          color: "#00ff00",
          padding: "4px 8px",
          fontSize: "12px",
          fontFamily: "monospace",
          zIndex: 9999,
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
          gap: "16px",
        }}
      >
        <span style={{ color: searchParams.get("width") ? "#ffff00" : "#00ff00" }}>
          W: {searchParams.get("width") || windowSize.width}px
        </span>
        <span style={{ color: searchParams.get("height") ? "#ffff00" : "#00ff00" }}>
          H: {searchParams.get("height") || windowSize.height}px
        </span>
        <span>Ratio: {(windowSize.height / windowSize.width).toFixed(2)}</span>
        <span>{window.location.pathname === "/mobile" ? "Mobile Mode" : "Desktop Mode"}</span>
      </Box>
      {/* 上段: Live2D Model (高さ768px = 60%) */}
      <Box
        className={styles.live2dSection || ""}
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          padding: "4px",
          boxSizing: "border-box",
          position: "relative",
        }}
      >
        <Live2DModelViewer
          width={380}
          height={700}
          speechText={speechText}
          isSpeaking={showSpeechBubble}
          bubbleSide="bottom"
          useCard={false}
          bubbleMaxWidth={360}
          specifiedWidth={specifiedDimensions.width}
          bubbleOffsetY={-30}
          {...(currentAudioData
            ? { audioData: currentAudioData, audioMessageId: currentMessageId }
            : {})}
          onAudioEnd={handleAudioEnd}
        />
      </Box>

      {/* 下段: Agent List (最大2つ) と Chat Component */}
      <Box
        className={styles.chatSection || ""}
        style={{
          borderTop: "1px solid rgba(255, 255, 255, 0.1)",
          boxSizing: "border-box",
          display: "flex",
          flexDirection: "column",
          gap: "4px",
        }}
      >
        {/* Agent List - コンパクト表示（最大2つ） */}
        <Box
          style={{
            flex: "0 0 auto",
            padding: "4px",
            borderBottom: "1px solid rgba(255, 255, 255, 0.1)",
          }}
        >
          <MobileAgentList
            onAgentClick={(agent) => {
              // Toggle agent selection
              if (selectedAgent?.pid === agent.pid) {
                // Clear agent mode
                setSelectedAgent(null);
                messageRouter.clearMode();
                wsClient.current?.clearAgentMode();
              } else {
                // Set agent mode
                setSelectedAgent(agent);
                messageRouter.setAgentMode(agent);
                wsClient.current?.setAgentMode(agent.pid);
              }
            }}
            selectedAgentPID={selectedAgent?.pid ?? null}
            wsClient={wsClient.current}
            maxAgents={2}
          />
        </Box>

        {/* Chat Display */}
        <Box
          style={{
            flex: 1,
            minHeight: 0,
            overflow: "hidden",
          }}
        >
          <ChatDisplay
            wsClient={wsClient.current}
            connectionStatus={connectionStatus}
            currentPlayingMessageId={currentMessageId}
            variant="mobile"
            maxDisplayMessages={3}
            showInput={false}
            onAudioToggle={handleAudioToggle}
            isAudioEnabled={isAudioEnabled}
            agentPID={selectedAgent?.pid ?? null}
            onAgentDisconnect={() => setSelectedAgent(null)}
          />
        </Box>
      </Box>
    </Box>
  );
};
