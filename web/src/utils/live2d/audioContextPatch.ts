interface SoundManagerType {
  audios?: HTMLAudioElement[];
  contexts?: AudioContext[];
  addContext?(audio: HTMLAudioElement): AudioContext;
  dispose?(audio: HTMLAudioElement): void;
  _isPatched?: boolean;
}

export function patchSoundManager(SoundManager: SoundManagerType): AudioContext | null {
  if (!SoundManager) {
    console.error("[patchSoundManager] SoundManager is null or undefined");
    return null;
  }

  // 既にパッチ済みかチェック
  try {
    if (SoundManager._isPatched) {
      return sharedAudioContextInstance;
    }
  } catch {
    // _isPatchedが追加できない場合は続行
  }

  let sharedAudioContext: AudioContext | null = null;
  const patchedAudioContextWeakMap = new WeakMap<HTMLAudioElement, AudioContext>();

  // オブジェクトが拡張可能かチェック
  if (!Object.isExtensible(SoundManager)) {
    console.warn("[patchSoundManager] SoundManager is not extensible, skipping patch");
    // パッチできない場合でも、共有AudioContextを作成して返す
    if (!sharedAudioContextInstance) {
      sharedAudioContextInstance = new AudioContext();
    }
    return sharedAudioContextInstance;
  }

  // contextsとaudiosの配列を確保
  try {
    if (!SoundManager.contexts) {
      SoundManager.contexts = [];
    }
    if (!SoundManager.audios) {
      SoundManager.audios = [];
    }
  } catch (error) {
    console.warn("[patchSoundManager] Cannot add properties to SoundManager:", error);
    // プロパティを追加できない場合でも続行
  }

  const originalDispose = SoundManager.dispose?.bind(SoundManager);

  if (!originalDispose) {
    console.error("[patchSoundManager] SoundManager.dispose not found");
    return null;
  }

  try {
    SoundManager.addContext = function (audio: HTMLAudioElement): AudioContext {
      if (!sharedAudioContext || sharedAudioContext.state === "closed") {
        sharedAudioContext = new AudioContext();
      }

      patchedAudioContextWeakMap.set(audio, sharedAudioContext);

      if (this.contexts && !this.contexts.includes(sharedAudioContext)) {
        this.contexts.push(sharedAudioContext);
      }

      return sharedAudioContext;
    };

    SoundManager.dispose = function (audio: HTMLAudioElement): void {
      const context = patchedAudioContextWeakMap.get(audio);

      if (context === sharedAudioContext) {
        patchedAudioContextWeakMap.delete(audio);

        if (this.contexts) {
          const index = this.contexts.indexOf(context);
          if (index > -1) {
            this.contexts.splice(index, 1);
          }
        }

        audio.pause();
        audio.removeAttribute("src");

        if (this.audios) {
          const audioIndex = this.audios.indexOf(audio);
          if (audioIndex > -1) {
            this.audios.splice(audioIndex, 1);
          }
        }
      } else {
        originalDispose?.call(this, audio);
      }
    };

    window.addEventListener("beforeunload", () => {
      if (sharedAudioContext && sharedAudioContext.state !== "closed") {
        sharedAudioContext.close();
      }
    });

    // パッチ済みフラグを設定
    try {
      SoundManager._isPatched = true;
    } catch {
      // _isPatchedが追加できない場合は無視
    }

    // グローバル変数に保存（デバッグ用）
    sharedAudioContextInstance = sharedAudioContext;

    isPatchApplied = true;

    return sharedAudioContext;
  } catch (error) {
    console.error("[patchSoundManager] Failed to apply patch:", error);
    // パッチが失敗しても共有AudioContextを返す
    if (!sharedAudioContextInstance) {
      sharedAudioContextInstance = new AudioContext();
    }
    return sharedAudioContextInstance;
  }
}

let isPatchApplied = false;
let sharedAudioContextInstance: AudioContext | null = null;

export interface SoundManagerPatchStatus {
  isPatchApplied: boolean;
  sharedAudioContext: AudioContext | null;
  audioContextCount: number;
  patchMethod: string | null;
}

/**
 * 共有AudioContextを取得する
 */
export function getSharedAudioContext(): AudioContext | null {
  return sharedAudioContextInstance;
}

/**
 * 共有AudioContextをresumeする
 */
export async function resumeSharedAudioContext(): Promise<void> {
  if (sharedAudioContextInstance && sharedAudioContextInstance.state === "suspended") {
    await sharedAudioContextInstance.resume();
  }
}

export function getSoundManagerPatchStatus(): SoundManagerPatchStatus {
  // PIXI.live2d.SoundManagerの状態を確認
  const PIXI = window.PIXI;
  let audioContextCount = 0;

  if (PIXI?.live2d?.SoundManager?.contexts) {
    audioContextCount = PIXI.live2d.SoundManager.contexts.length;
  }

  return {
    isPatchApplied,
    sharedAudioContext: sharedAudioContextInstance,
    audioContextCount,
    patchMethod: isPatchApplied ? "SoundManager.addContext/dispose override" : null,
  };
}

/**
 * SoundManagerを探して返す
 * @param PixiLive2D - pixi-live2d-display-lipsyncpatchモジュール
 * @param Live2DModel - Live2DModelクラス（オプション）
 * @returns 見つかったSoundManager、見つからない場合はnull
 */
export function findSoundManager(
  PixiLive2D?: unknown,
  Live2DModel?: unknown,
): SoundManagerType | null {
  let SoundManager: SoundManagerType | null = null;

  // 1. 直接エクスポートされている場合
  const pixiModule = PixiLive2D as { SoundManager?: SoundManagerType };
  if (pixiModule?.SoundManager) {
    SoundManager = pixiModule.SoundManager;
    return SoundManager;
  }

  // 2. defaultエクスポートの中にある場合
  const pixiWithDefault = PixiLive2D as { default?: { SoundManager?: SoundManagerType } };
  if (pixiWithDefault?.default) {
    const defaultExport = pixiWithDefault.default;
    if (defaultExport.SoundManager) {
      SoundManager = defaultExport.SoundManager;
      return SoundManager;
    }
  }

  // 3. Live2DModelクラスから取得
  const modelClass = Live2DModel as { SoundManager?: SoundManagerType };
  if (modelClass?.SoundManager) {
    SoundManager = modelClass.SoundManager;
    return SoundManager;
  }

  // 4. PIXI.live2dから取得（既に設定されている場合）
  if (window.PIXI?.live2d?.SoundManager) {
    SoundManager = window.PIXI.live2d.SoundManager;
    return SoundManager;
  }

  // 5. モジュール全体を探索
  if (PixiLive2D && typeof PixiLive2D === "object") {
    const pixiObj = PixiLive2D as Record<string, unknown>;
    for (const key in pixiObj) {
      const value = pixiObj[key] as { SoundManager?: SoundManagerType };
      if (value && typeof value === "object" && value.SoundManager) {
        SoundManager = value.SoundManager;
        return SoundManager;
      }
    }
  }

  return null;
}

/**
 * SoundManagerを探してパッチを適用する便利関数
 * @param PixiLive2D - pixi-live2d-display-lipsyncpatchモジュール
 * @param Live2DModel - Live2DModelクラス（オプション）
 * @returns パッチが適用された場合はAudioContext、失敗した場合はnull
 */
export function findAndPatchSoundManager(
  PixiLive2D?: unknown,
  Live2DModel?: unknown,
): AudioContext | null {
  const SoundManager = findSoundManager(PixiLive2D, Live2DModel);

  if (SoundManager) {
    // まず通常のパッチを試みる
    const context = patchSoundManager(SoundManager);

    // パッチが失敗した場合、プロキシラッパーを使用
    if (!context && !Object.isExtensible(SoundManager)) {
      console.warn("[findAndPatchSoundManager] Using proxy wrapper for frozen SoundManager");
      const patchedContext = createPatchedSoundManagerProxy(SoundManager);

      // PIXIにプロキシを設定
      if (window.PIXI) {
        if (!window.PIXI.live2d) {
          window.PIXI.live2d = {};
        }
        window.PIXI.live2d.SoundManager = patchedContext.proxy;
      }

      // Live2DModelクラスのSoundManagerも置き換える
      if (Live2DModel && typeof Live2DModel === "function") {
        try {
          const modelConstructor = Live2DModel as { SoundManager?: SoundManagerType };
          if (modelConstructor.SoundManager) {
            modelConstructor.SoundManager = patchedContext.proxy;
          }
        } catch (e) {
          console.warn("Could not replace Live2DModel.SoundManager:", e);
        }
      }

      return patchedContext.context;
    }

    // PIXIにも設定
    if (window.PIXI) {
      if (!window.PIXI.live2d) {
        window.PIXI.live2d = {};
      }
      window.PIXI.live2d.SoundManager = SoundManager;
    }

    return context;
  } else {
    console.error(
      "[findAndPatchSoundManager] SoundManager not found - audio may not work correctly",
    );
    return null;
  }
}

/**
 * frozenなSoundManagerのためのプロキシラッパーを作成
 */
function createPatchedSoundManagerProxy(OriginalSoundManager: SoundManagerType): {
  proxy: SoundManagerType;
  context: AudioContext;
} {
  let sharedAudioContext: AudioContext | null = null;
  const patchedAudioContextWeakMap = new WeakMap<HTMLAudioElement, AudioContext>();

  // 共有AudioContextを作成
  if (!sharedAudioContextInstance) {
    sharedAudioContextInstance = new AudioContext();
  }
  sharedAudioContext = sharedAudioContextInstance;

  // プロキシハンドラー
  const handler: ProxyHandler<SoundManagerType> = {
    get(target, prop, receiver) {
      // addContextメソッドをオーバーライド
      if (prop === "addContext") {
        return (audio: HTMLAudioElement): AudioContext => {
          if (!sharedAudioContext || sharedAudioContext.state === "closed") {
            sharedAudioContext = new AudioContext();
            sharedAudioContextInstance = sharedAudioContext;
          }

          patchedAudioContextWeakMap.set(audio, sharedAudioContext);

          // 元のcontexts配列にも追加を試みる（可能な場合）
          try {
            if (target.contexts && !target.contexts.includes(sharedAudioContext)) {
              target.contexts.push(sharedAudioContext);
            }
          } catch {
            // 追加できない場合は無視
          }

          return sharedAudioContext;
        };
      }

      // disposeメソッドをオーバーライド
      if (prop === "dispose") {
        return (audio: HTMLAudioElement): void => {
          const context = patchedAudioContextWeakMap.get(audio);

          if (context === sharedAudioContext) {
            patchedAudioContextWeakMap.delete(audio);

            // 元のcontexts配列からも削除を試みる
            try {
              if (target.contexts) {
                const index = target.contexts.indexOf(context);
                if (index > -1) {
                  target.contexts.splice(index, 1);
                }
              }
            } catch {
              // 削除できない場合は無視
            }

            audio.pause();
            audio.removeAttribute("src");

            // 元のaudios配列からも削除を試みる
            try {
              if (target.audios) {
                const audioIndex = target.audios.indexOf(audio);
                if (audioIndex > -1) {
                  target.audios.splice(audioIndex, 1);
                }
              }
            } catch {
              // 削除できない場合は無視
            }
          } else if (target.dispose) {
            // 元のdisposeメソッドを呼ぶ
            target.dispose.call(target, audio);
          }
        };
      }

      // _isPatchedプロパティ
      if (prop === "_isPatched") {
        return true;
      }

      // その他のプロパティは元のオブジェクトから取得
      return Reflect.get(target, prop, receiver);
    },

    set(target, prop, value) {
      // frozenオブジェクトへの書き込みは無視
      if (prop === "_isPatched") {
        return true;
      }
      try {
        return Reflect.set(target, prop, value);
      } catch {
        return true; // エラーを無視
      }
    },
  };

  const proxy = new Proxy(OriginalSoundManager, handler);

  isPatchApplied = true;

  return { proxy, context: sharedAudioContext };
}

export function ensureSoundManagerPatch(): void {
  if (isPatchApplied) {
    return;
  }

  try {
    // PIXI.live2d.SoundManagerを探す
    const SoundManager = findSoundManager();

    if (SoundManager) {
      const context = patchSoundManager(SoundManager);
      if (context) {
        isPatchApplied = true;
      }
    }
  } catch (error) {
    console.error("[ensureSoundManagerPatch] Failed to patch SoundManager:", error);
  }
}
