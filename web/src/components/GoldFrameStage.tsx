import type React from "react";
import { useId } from "react";

type Props = {
  title?: string;
  width?: number; // カード幅（px）
  lineW?: number; // 線の太さ（外枠/内枠/ノッチ共通）
  className?: string;
  children?: React.ReactNode; // img/canvas/video など
};

/**
 * 二重の金縁（外：角丸長方形 / 内：凹ノッチを含む“1本のパス”）＋ 上部プレート（一重）
 * - viewBox: 1000x1333 を基準に、拡縮しても線幅は non-scaling-stroke で安定
 */
const GoldFrameStage: React.FC<Props> = ({
  title = "ASSISTANT",
  width = 400,
  lineW = 4,
  className,
  children,
}) => {
  const id = useId();
  const goldGradientId = `${id}-gold`;
  const goldThinGradientId = `${id}-goldThin`;
  const bgGradientId = `${id}-bgGradient`;

  // --- 基本寸法（SVG座標） ---
  // canvas: 360x480
  // フレームのマージン: 16px
  // タイトルフレームのために上部に余白を追加
  const VBW = 392,
    VBH = 540; // 高さを増やしてタイトルフレームを収める

  // 外枠（外側の金線）- 下にオフセットしてタイトルフレームのスペースを確保
  const frameOffsetY = 20; // フレーム全体を下にオフセット
  const outer = { x: 0, y: frameOffsetY, w: VBW, h: 512, r: 24 };

  // 外枠（内側の金線 - 二重線の内側）
  const outerInner = { x: 6, y: frameOffsetY + 6, w: VBW - 12, h: 500, r: 22 };

  // 内枠（背景エリア）- フレームに近づける
  const inner = { x: 10, y: frameOffsetY + 10, w: VBW - 20, h: 492, r: 20 };

  function buildConcaveInnerPathD({
    x,
    y,
    w,
    h,
    R, // スクープ（内側にえぐれた角）の半径
  }: {
    x: number;
    y: number;
    w: number;
    h: number;
    R: number;
  }) {
    const left = x;
    const right = x + w;
    const top = y;
    const bottom = y + h;

    const d: string[] = [];

    // 左上から開始（上辺の開始点）
    d.push(`M ${left + R},${top}`);

    // 上辺（右へ）
    d.push(`H ${right - R}`);

    // 右上角（スクープ：内側にえぐれる）
    // 中心点は (right, top) で、半径Rの円弧を反時計回りに描く
    d.push(`A ${R} ${R} 0 0 0 ${right},${top + R}`);

    // 右辺（下へ）
    d.push(`V ${bottom - R}`);

    // 右下角（スクープ）
    d.push(`A ${R} ${R} 0 0 0 ${right - R},${bottom}`);

    // 下辺（左へ）
    d.push(`H ${left + R}`);

    // 左下角（スクープ）
    d.push(`A ${R} ${R} 0 0 0 ${left},${bottom - R}`);

    // 左辺（上へ）
    d.push(`V ${top + R}`);

    // 左上角（スクープ）
    d.push(`A ${R} ${R} 0 0 0 ${left + R},${top}`);

    // 閉じる
    d.push(`Z`);

    return d.join(" ");
  }

  return (
    <div
      className={`gold-frame-stage ${className ?? ""}`}
      style={{ "--card-w": `${width}px` } as React.CSSProperties}
    >
      <div className="gold-stage">
        <svg
          className="gold-frame"
          viewBox={`0 0 ${VBW} ${VBH}`}
          preserveAspectRatio="none"
          aria-hidden="true"
        >
          <defs>
            {/* 金グラデ（外枠/プレート用） */}
            <linearGradient id={goldGradientId} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0" stopColor="#6b5120" />
              <stop offset="0.25" stopColor="#d2b05a" />
              <stop offset="0.5" stopColor="#f6e6b5" />
              <stop offset="0.75" stopColor="#d2b05a" />
              <stop offset="1" stopColor="#6b5120" />
            </linearGradient>
            {/* 内側（凹ノッチ含む）細線用 */}
            <linearGradient id={goldThinGradientId} x1="0" y1="0" x2="1" y2="0">
              <stop offset="0" stopColor="#6b5120" />
              <stop offset="0.5" stopColor="#e9d79f" />
              <stop offset="1" stopColor="#6b5120" />
            </linearGradient>
            {/* 背景のグラデーション（紫、下が明るい） */}
            <linearGradient id={bgGradientId} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0" stopColor="#3d3b66" />
              <stop offset="1" stopColor="#5a5682" />
            </linearGradient>
          </defs>

          {/* 背景エリア（スクープ状の角） */}
          <path
            d={buildConcaveInnerPathD({
              x: inner.x,
              y: inner.y,
              w: inner.w,
              h: inner.h,
              R: inner.r,
            })}
            fill={`url(#${bgGradientId})`}
            stroke="none"
          />

          {/* 外側の枠 - 外線（スクープ状の角） */}
          <path
            d={buildConcaveInnerPathD({
              x: outer.x,
              y: outer.y,
              w: outer.w,
              h: outer.h,
              R: outer.r,
            })}
            fill="none"
            stroke={`url(#${goldGradientId})`}
            strokeWidth={lineW * 0.5}
            strokeLinejoin="round"
            strokeLinecap="round"
            vectorEffect="non-scaling-stroke"
          />

          {/* 外側の枠 - 内線（スクープ状の角、二重線） */}
          <path
            d={buildConcaveInnerPathD({
              x: outerInner.x,
              y: outerInner.y,
              w: outerInner.w,
              h: outerInner.h,
              R: outerInner.r,
            })}
            fill="none"
            stroke={`url(#${goldThinGradientId})`}
            strokeWidth={lineW * 0.6}
            strokeLinejoin="round"
            strokeLinecap="round"
            vectorEffect="non-scaling-stroke"
          />

          {/* 上部プレート（背景と金縁の組み合わせで黒い線を表現） */}
          <g className="title">
            {/* プレートの座標を変数化 */}
            {(() => {
              const plateW = 200;
              const plateH = 50;
              const plateX = VBW / 2 - plateW / 2; // 正確に中央に配置
              const plateY = 10; // メインフレームに重なる位置
              const plateCenterX = plateX + plateW / 2;
              const plateCenterY = plateY + plateH / 2;

              return (
                <>
                  {/* 1層目：黒い背景（縁なし） */}
                  <path
                    d={buildConcaveInnerPathD({
                      x: plateX,
                      y: plateY,
                      w: plateW,
                      h: plateH,
                      R: 15,
                    })}
                    fill="rgba(0, 0, 0, 0.95)"
                    stroke="none"
                  />

                  {/* 2層目：薄い金の縁（背景なし） */}
                  <path
                    d={buildConcaveInnerPathD({
                      x: plateX + 3,
                      y: plateY + 3,
                      w: plateW - 6,
                      h: plateH - 6,
                      R: 14,
                    })}
                    fill="none"
                    stroke={`url(#${goldThinGradientId})`}
                    strokeWidth={1.5}
                    vectorEffect="non-scaling-stroke"
                  />

                  {/* 3層目：黒い背景（金縁の内側、黒い線として見える） */}
                  <path
                    d={buildConcaveInnerPathD({
                      x: plateX + 8,
                      y: plateY + 8,
                      w: plateW - 16,
                      h: plateH - 16,
                      R: 13,
                    })}
                    fill="rgba(0, 0, 0, 0.9)"
                    stroke="none"
                  />

                  {/* 4層目：薄暗い背景 */}
                  <path
                    d={buildConcaveInnerPathD({
                      x: plateX + 8,
                      y: plateY + 8,
                      w: plateW - 16,
                      h: plateH - 16,
                      R: 13,
                    })}
                    fill="rgba(30, 28, 45, 0.9)"
                    stroke="none"
                  />

                  {/* 5層目：最内側の薄い金の縁 */}
                  <path
                    d={buildConcaveInnerPathD({
                      x: plateX + 7,
                      y: plateY + 7,
                      w: plateW - 14,
                      h: plateH - 14,
                      R: 13,
                    })}
                    fill="none"
                    stroke={`url(#${goldThinGradientId})`}
                    strokeWidth={1}
                    vectorEffect="non-scaling-stroke"
                  />

                  {/* テキスト */}
                  <text
                    x={plateCenterX}
                    y={plateCenterY}
                    textAnchor="middle"
                    dominantBaseline="middle" // 垂直方向の中央揃え
                    fontSize={20}
                    fontWeight={500}
                    letterSpacing={3}
                    fill={`url(#${goldGradientId})`}
                    fontFamily={`"Cinzel", "Marcellus SC", "EB Garamond", serif`}
                  >
                    {title}
                  </text>
                </>
              );
            })()}
          </g>
        </svg>

        {/* Live2Dモデルなどのコンテンツ（背景の上に配置） */}
        {children}
      </div>

      {/* スタイル（カード本体・ステージ） */}
      <style>{`
        :root{
          --panel-grad1: #3d3b66;
          --panel-grad2: #5a5682;
          --card-bg: #0b0d11;
        }
        .gold-frame-stage{
          width: 440px;  /* 固定幅440px */
          height: 540px; /* 固定高さ540px */
          border-radius: 24px;
          background: var(--card-bg);  /* 黒い背景を復元 */
          box-shadow:
            0 18px 48px rgba(0,0,0,.35),
            inset 0 2px 0 rgba(255,255,255,.06),
            inset 0 -3px 10px rgba(0,0,0,.55);
          overflow: visible; /* SVGが外側に表示されるように */
          position: relative; 
          isolation: isolate;
          display: flex;
          justify-content: center;
          align-items: center;
        }
        .gold-stage{
          position: relative;
          margin: 0;  /* マージンを削除 */
          width: 392px;  /* フレームの幅 */
          height: 540px; /* tcg-cardの高さに合わせる */
          overflow: hidden;  /* Live2Dモデルを枠内に収める */
          border-radius: 20px;  /* スクープ形状に合わせる */
        }
        .gold-art{ 
          position:absolute; 
          inset:0; 
          display:flex; 
          justify-content:center; 
          align-items:center;
          padding: 40px 0px 0px 0px;  /* タイトルフレームの分だけ余白 */
          pointer-events: auto;  /* インタラクション可能に */
        }
        .gold-frame{ 
          position:absolute; 
          inset:0; 
          width:100%; 
          height:100%; 
          pointer-events:none; 
          z-index: 1;  /* 背景レイヤー */
        }
      `}</style>
    </div>
  );
};

export default GoldFrameStage;
