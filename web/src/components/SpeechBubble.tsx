import { Box } from "@mantine/core";
import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import "./SpeechBubble.css";

interface SpeechBubbleProps {
  text: string;
  visible: boolean;
  anchorSelector?: string;
  side?: "right" | "left" | "top" | "bottom";
  withWave?: boolean;
  typewriter?: boolean;
  style?: React.CSSProperties;
  maxWidth?: number;
  isMobile?: boolean;
  specifiedWidth?: number;
}

export function SpeechBubble({
  text,
  visible,
  anchorSelector,
  side = "right",
  withWave = true,
  typewriter = true,
  style = {},
  maxWidth,
  isMobile = false,
  specifiedWidth,
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
      const viewportWidth = window.innerWidth;
      const gap = 20;

      // /mobileパスの場合は必ずモバイルUIとして扱う
      const isInMobileView = window.location.pathname === "/mobile" || isMobile;

      // モバイルビューでは吹き出し幅を調整
      let bubbleWidth: number;
      if (isInMobileView) {
        // モバイルの場合、画面幅に応じて吹き出し幅を調整
        bubbleWidth = maxWidth || Math.min(360, viewportWidth - 40);
      } else {
        bubbleWidth = maxWidth || Math.min(600, viewportWidth - 40);
      }

      // 画面上の絶対位置で配置
      let positions: Record<string, { left: number; top: number }>;

      if (isInMobileView) {
        // モバイルの場合
        // チャット欄は画面の60%の位置（768px）から始まる
        // Live2Dエリアの下部に吹き出しを配置（チャット欄の上に余裕を持って）
        const viewportHeight = window.innerHeight;

        // 指定された幅がある場合はそれを使用、なければ実際の表示幅を取得
        let mobileContainerWidth: number;
        if (specifiedWidth) {
          // URLパラメータで指定された幅を使用
          mobileContainerWidth = specifiedWidth;
        } else {
          // 実際の表示幅を取得（document.documentElement.clientWidthを使用）
          const actualDisplayWidth = document.documentElement.clientWidth;
          mobileContainerWidth = Math.min(400, actualDisplayWidth);
        }

        // 実際の画面サイズに基づいて位置を計算
        // チャット欄開始位置（768px）より上、Live2Dエリアの下部に配置
        // 1280pxの画面で768pxは60%、その少し上（50-55%あたり）に配置
        const bubbleTop = Math.min(680, viewportHeight * 0.53); // 画面高さの53%（約680px）

        // 横幅の中央配置を計算（指定幅または実際の表示幅を基準に）
        // 指定幅がある場合は、その幅の中央に配置
        let centerLeft: number;
        if (specifiedWidth) {
          // 指定幅がある場合、実際のviewport幅に関係なく
          // 指定幅の中央に吹き出しを配置
          // 例: 指定幅400px、吹き出し360pxの場合、(400-360)/2 = 20px
          centerLeft = (specifiedWidth - bubbleWidth) / 2;
        } else {
          // 通常の計算
          const containerCenterX =
            (viewportWidth - mobileContainerWidth) / 2 + mobileContainerWidth / 2;
          centerLeft = containerCenterX - bubbleWidth / 2;
        }

        // デバッグ情報をコンソールに出力
        console.log("Mobile Bubble Position:", {
          viewportWidth,
          specifiedWidth,
          mobileContainerWidth,
          viewportHeight,
          bubbleWidth,
          bubbleTop,
          centerLeft,
          isInMobileView,
          pathname: window.location.pathname,
        });

        positions = {
          right: { left: centerLeft, top: bubbleTop },
          left: { left: centerLeft, top: bubbleTop },
          top: { left: centerLeft, top: bubbleTop },
          bottom: { left: centerLeft, top: bubbleTop },
        };
      } else {
        positions = {
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
      }

      const p = positions[side];

      // 画面端でのはみ出しを防ぐ
      const viewportHeight = window.innerHeight;
      let finalLeft = p?.left ?? 0;
      let finalTop = p?.top ?? 0;

      // モバイルビューの場合は端のチェックをスキップ
      if (!isInMobileView) {
        // 右端チェック
        if (finalLeft + bubbleWidth > viewportWidth - 20) {
          finalLeft = viewportWidth - bubbleWidth - 20;
        }
        // 左端チェック
        if (finalLeft < 20) {
          finalLeft = 20;
        }
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
  }, [anchorSelector, side, isMobile, maxWidth, specifiedWidth]);

  // モバイルビューの判定
  const isInMobileView = window.location.pathname === "/mobile" || isMobile;

  // Portalを使ってbody直下にレンダリング
  const bubbleContent = (
    <Box
      ref={ref}
      className={`speech-bubble-overlay ${visible ? "show" : "hide"} ${side}`}
      aria-live="polite"
      aria-atomic="true"
      style={style}
    >
      <div className="bubble">
        <span className="bubble-text">{typed}</span>
        {withWave && !isInMobileView && (
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
