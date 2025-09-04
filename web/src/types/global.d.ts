interface Window {
  PIXI: {
    live2d?: {
      SoundManager?: {
        audios?: HTMLAudioElement[];
        contexts?: AudioContext[];
        addContext?(audio: HTMLAudioElement): AudioContext;
        dispose?(audio: HTMLAudioElement): void;
        _isPatched?: boolean;
      };
      [key: string]: unknown;
    };
    [key: string]: unknown;
  };
  Live2D: unknown;
  Live2DCubismCore: unknown;
  audioContext?: AudioContext;
  webkitAudioContext?: typeof AudioContext;
  pendingAudioUrl?: string;
}

// pixi-live2d-display-lipsyncpatch の型定義
declare module "pixi-live2d-display-lipsyncpatch/cubism4" {
  export const SoundManager: {
    audios: HTMLAudioElement[];
    contexts: AudioContext[];
    addContext(audio: HTMLAudioElement): AudioContext;
    dispose(audio: HTMLAudioElement): void;
    _pwaPatched?: boolean;
    _isPatched?: boolean;
  };

  export class Live2DModel {
    static from(source: string | object, options?: unknown): Promise<Live2DModel>;
    static fromSync(source: string | object, options?: unknown): Live2DModel;

    internalModel: {
      coreModel?: unknown;
      focusController?: {
        focus(x: number, y: number, instant?: boolean): void;
      };
      motionManager?: {
        motionGroups?: Record<string, unknown[]>;
        definitions?: Record<string, unknown[]>;
      };
      expressionManager?: unknown;
      settings?: unknown;
    };

    x: number;
    y: number;
    scale: {
      set(value: number): void;
      x: number;
      y: number;
    };

    destroy(): void;
    motion(group: string, index?: number, priority?: number): Promise<void>;
    expression(name: string): Promise<void>;
    speak(audio: string | HTMLAudioElement, options?: unknown): void;
    stopSpeaking(): void;
    focus(x: number, y: number, instant?: boolean): void;
    getLocalBounds(): { x: number; y: number; width: number; height: number };

    on(event: string, callback: (hitAreas: string[]) => void): void;
    off(event: string, callback?: (hitAreas: string[]) => void): void;
  }
}
