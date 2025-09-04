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
  if (SoundManager._isPatched) {
    return sharedAudioContextInstance;
  }

  let sharedAudioContext: AudioContext | null = null;
  const patchedAudioContextWeakMap = new WeakMap<HTMLAudioElement, AudioContext>();

  // contextsとaudiosの配列を確保
  if (!SoundManager.contexts) {
    SoundManager.contexts = [];
  }
  if (!SoundManager.audios) {
    SoundManager.audios = [];
  }

  const originalDispose = SoundManager.dispose?.bind(SoundManager);

  if (!originalDispose) {
    console.error("[patchSoundManager] SoundManager.dispose not found");
    return null;
  }

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
  SoundManager._isPatched = true;

  // グローバル変数に保存（デバッグ用）
  sharedAudioContextInstance = sharedAudioContext;

  isPatchApplied = true;

  return sharedAudioContext;
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
    const context = patchSoundManager(SoundManager);

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
