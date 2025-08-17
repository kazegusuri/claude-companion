import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Box } from "@mantine/core";
import "./SpeechBubble.css";

interface SpeechBubbleProps {
  text: string;
  visible: boolean;
  anchorSelector?: string;
  side?: "right" | "left" | "top" | "bottom";
  withWave?: boolean;
  typewriter?: boolean;
}

export function SpeechBubble({
  text,
  visible,
  anchorSelector,
  side = "right",
  withWave = true,
  typewriter = true,
}: SpeechBubbleProps) {
  const ref = useRef<HTMLDivElement>(null);
  const [typed, setTyped] = useState("");

  // 文字送りエフェクト
  useEffect(() => {
    if (!typewriter || !visible) {
      setTyped(text || "");
      return;
    }
    setTyped("");
    let i = 0;
    const id = setInterval(() => {
      i++;
      setTyped(text.slice(0, i));
      if (i >= (text?.length ?? 0)) clearInterval(id);
    }, 18);
    return () => clearInterval(id);
  }, [text, visible, typewriter]);

  // アンカーの位置に追従（固定配置）
  useEffect(() => {
    if (!ref.current || !anchorSelector) return;
    const anchor = document.querySelector(anchorSelector);
    if (!anchor) return;

    const updatePosition = () => {
      const el = ref.current;
      if (!el) return;

      // アンカー要素の画面上の位置を取得
      const rect = anchor.getBoundingClientRect();

      // 吹き出しのサイズ
      const bubbleWidth = 600; // 幅を少し広げる
      const gap = 20;

      // 画面上の絶対位置で配置
      const positions = {
        right: {
          left: rect.right + gap, // アンカーの右端から右へ
          top: rect.top + rect.height * 0.2, // 上から20%の位置
        },
        left: {
          left: rect.left - bubbleWidth - gap,
          top: rect.top + rect.height * 0.2,
        },
        top: {
          left: rect.left + (rect.width - bubbleWidth) / 2,
          top: rect.top - 150,
        },
        bottom: {
          left: rect.left + (rect.width - bubbleWidth) / 2,
          top: rect.bottom + gap,
        },
      };

      const p = positions[side];

      // 画面端でのはみ出しを防ぐ
      const viewportWidth = window.innerWidth;
      const viewportHeight = window.innerHeight;
      let finalLeft = p.left;
      let finalTop = p.top;

      // 右端チェック
      if (finalLeft + bubbleWidth > viewportWidth - 20) {
        finalLeft = viewportWidth - bubbleWidth - 20;
      }
      // 左端チェック
      if (finalLeft < 20) {
        finalLeft = 20;
      }
      // 下端チェック
      if (finalTop + 150 > viewportHeight - 20) {
        finalTop = viewportHeight - 170;
      }
      // 上端チェック
      if (finalTop < 20) {
        finalTop = 20;
      }

      Object.assign(el.style, {
        left: `${finalLeft}px`,
        top: `${finalTop}px`,
        width: `${bubbleWidth}px`,
      });
    };

    const ro = new ResizeObserver(updatePosition);
    ro.observe(anchor);
    window.addEventListener("resize", updatePosition);
    window.addEventListener("scroll", updatePosition, { passive: true });
    updatePosition();

    return () => {
      ro.disconnect();
      window.removeEventListener("resize", updatePosition);
      window.removeEventListener("scroll", updatePosition);
    };
  }, [anchorSelector, side, text, visible]);

  // Portalを使ってbody直下にレンダリング
  const bubbleContent = (
    <Box
      ref={ref}
      className={`speech-bubble-overlay ${visible ? "show" : "hide"} ${side}`}
      aria-live="polite"
      aria-atomic="true"
    >
      <div className="bubble">
        <span className="bubble-text">{typed}</span>
        {withWave && (
          <div className="bubble-wave" aria-hidden="true">
            <i />
            <i />
            <i />
            <i />
          </div>
        )}
      </div>
    </Box>
  );

  // document.bodyにPortalでレンダリング
  return createPortal(bubbleContent, document.body);
}
