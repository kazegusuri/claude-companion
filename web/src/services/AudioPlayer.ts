export interface PlaybackOptions {
  onStart?: () => void;
  onEnd?: () => void;
  onError?: (error: Error) => void;
}

export class AudioPlayer {
  private audioContext: AudioContext | null = null;
  private currentSource: AudioBufferSourceNode | null = null;
  private gainNode: GainNode | null = null;
  private isPlaying = false;
  private volume = 1.0;
  private isInitialized = false;

  constructor() {
    // Don't initialize context immediately, wait for first playback
  }

  // Public method to check if context is initialized
  isContextInitialized(): boolean {
    return this.isInitialized && this.audioContext !== null;
  }

  // Public method to check if context is suspended
  isContextSuspended(): boolean {
    return this.audioContext?.state === "suspended";
  }

  // Public method to manually initialize context (requires user interaction)
  async ensureInitialized(): Promise<void> {
    if (!this.isInitialized) {
      await this.initializeContext();
    }
  }

  private async initializeContext(): Promise<void> {
    if (this.audioContext) {
      return; // Already initialized
    }

    try {
      const AudioContextClass = window.AudioContext || (window as any).webkitAudioContext;
      this.audioContext = new AudioContextClass();

      // Check if context is suspended immediately after creation
      if (this.audioContext.state === "suspended") {
        console.log("AudioContext created in suspended state, attempting to resume...");
        try {
          await this.audioContext.resume();
          console.log("AudioContext resumed from suspended state");
        } catch (resumeError) {
          console.error("Failed to resume AudioContext:", resumeError);
          // If still suspended after attempting resume, clean up and throw error
          if (this.audioContext.state === "suspended") {
            console.error("AudioContext is still suspended - cleaning up");
            this.audioContext.close();
            this.audioContext = null;
            this.isInitialized = false;
            throw new Error("AudioContext cannot be activated - user interaction required");
          }
        }
      }

      // Create and connect gain node for volume control
      this.gainNode = this.audioContext.createGain();
      this.gainNode.connect(this.audioContext.destination);
      this.gainNode.gain.value = this.volume;

      this.isInitialized = true;
      console.log("AudioContext initialized successfully, state:", this.audioContext.state);
    } catch (error) {
      console.error("Failed to initialize AudioContext:", error);
      this.isInitialized = false;
      // Clean up any partially created context
      if (this.audioContext) {
        try {
          this.audioContext.close();
        } catch (closeError) {
          console.error("Failed to close AudioContext:", closeError);
        }
        this.audioContext = null;
      }
      throw error;
    }
  }

  async playBase64Audio(base64Data: string, options?: PlaybackOptions): Promise<void> {
    // Initialize context if not already done (lazy initialization)
    if (!this.audioContext || !this.gainNode) {
      try {
        await this.initializeContext();
        // Successfully initialized, call onStart
        options?.onStart?.();
      } catch (error) {
        // Skip audio playback if initialization fails (likely due to autoplay policy)
        console.warn(
          "Skipping audio playback - AudioContext not initialized. User interaction required.",
        );
        options?.onEnd?.(); // Call onEnd to continue queue processing
        return; // Skip playback instead of throwing error
      }
    } else {
      // Already initialized, call onStart
      options?.onStart?.();
    }

    try {
      // Stop any currently playing audio
      this.stop();

      // Resume audio context if suspended (required for some browsers)
      if (this.audioContext && this.audioContext.state === "suspended") {
        await this.audioContext.resume();
      }

      // Decode base64 to ArrayBuffer
      const audioData = this.base64ToArrayBuffer(base64Data);

      // Decode audio data
      const audioBuffer = await this.audioContext.decodeAudioData(audioData);

      // Create and configure source
      this.currentSource = this.audioContext.createBufferSource();
      this.currentSource.buffer = audioBuffer;
      this.currentSource.connect(this.gainNode);

      // Set up event handlers
      this.currentSource.onended = () => {
        this.isPlaying = false;
        this.currentSource = null;
        options?.onEnd?.();
      };

      // Start playback
      this.isPlaying = true;
      this.currentSource.start(0);
    } catch (error) {
      this.isPlaying = false;
      const err = error instanceof Error ? error : new Error("Audio playback failed");
      options?.onError?.(err);
      throw err;
    }
  }

  private base64ToArrayBuffer(base64: string): ArrayBuffer {
    // Remove data URL prefix if present
    const base64Clean = base64.replace(/^data:audio\/[a-z]+;base64,/, "");

    // Decode base64
    const binaryString = atob(base64Clean);
    const bytes = new Uint8Array(binaryString.length);

    for (let i = 0; i < binaryString.length; i++) {
      bytes[i] = binaryString.charCodeAt(i);
    }

    return bytes.buffer;
  }

  stop(): void {
    if (this.currentSource) {
      try {
        this.currentSource.stop();
        this.currentSource.disconnect();
      } catch (error) {
        // Ignore errors when stopping already stopped source
      }
      this.currentSource = null;
      this.isPlaying = false;
    }
  }

  setVolume(value: number): void {
    // Clamp volume between 0 and 1
    this.volume = Math.max(0, Math.min(1, value));

    if (this.gainNode) {
      // Use exponential ramp for smooth volume changes
      this.gainNode.gain.setTargetAtTime(this.volume, this.audioContext?.currentTime || 0, 0.015);
    }
  }

  getVolume(): number {
    return this.volume;
  }

  isCurrentlyPlaying(): boolean {
    return this.isPlaying;
  }

  async resume(): Promise<void> {
    if (this.audioContext?.state === "suspended") {
      await this.audioContext.resume();
    }
  }

  suspend(): void {
    if (this.audioContext?.state === "running") {
      this.audioContext.suspend();
    }
  }

  getState(): AudioContextState | null {
    return this.audioContext?.state || null;
  }
}
