export interface BroadcastChannelLike {
  onmessage: ((event: { data: unknown }) => void) | null;
  postMessage(message: unknown): void;
  close(): void;
}

export type TabClaimer = (token: string, tabId: string) => Promise<void>;
let defaultTabClaimer: TabClaimer | null = null;
export function setTabClaimer(claimer: TabClaimer | null): void {
  defaultTabClaimer = claimer;
}

export async function claimTabIfConfigured(token: string, tabId: string): Promise<void> {
  await defaultTabClaimer?.(token, tabId);
}

const TAB_CHANNEL_NAME = "gu-gameplay-tab";

function randomUUID(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

function readStorage(storageName: "localStorage" | "sessionStorage", key: string): string | null {
  if (typeof window === "undefined") return null;
  try {
    return window[storageName].getItem(key);
  } catch {
    return null;
  }
}

function writeStorage(storageName: "localStorage" | "sessionStorage", key: string, value: string): void {
  if (typeof window === "undefined") return;
  try {
    window[storageName].setItem(key, value);
  } catch {
    // ignore
  }
}

function defaultBroadcastChannel(name: string): BroadcastChannelLike | null {
  if (typeof window === "undefined" || typeof BroadcastChannel === "undefined") {
    return null;
  }
  try {
    return new BroadcastChannel(name) as unknown as BroadcastChannelLike;
  } catch {
    return null;
  }
}

/**
 * Duplicate-tab guard: Chrome "Duplicate tab" copies sessionStorage,
 * causing two live tabs to share `gu_gameplay_tab_id`. Each runtime instance gets
 * a unique in-memory `instanceId` and probes on BroadcastChannel; if another live
 * tab responds with the same `tabId`, the new tab rotates `tabId`.
 */
export class TabIdCoordinator {
  private readonly instanceId: string;
  private readonly getTabId: () => string | null;
  private readonly setTabId: (tabId: string) => void;
  private readonly generateId: () => string;
  private channel: BroadcastChannelLike | null = null;
  private conflictDetected = false;
  private readonly conflictListeners = new Set<(rotatedTabId: string) => void>();

  constructor(options: {
    instanceId?: string;
    getTabId?: () => string | null;
    setTabId?: (tabId: string) => void;
    generateId?: () => string;
    createBroadcastChannel?: (name: string) => BroadcastChannelLike | null;
    onConflictRotated?: (rotatedTabId: string) => void;
  } = {}) {
    this.generateId = options.generateId ?? randomUUID;
    this.instanceId = options.instanceId ?? this.generateId();
    this.getTabId = options.getTabId ?? (() => readStorage("sessionStorage", "gu_gameplay_tab_id"));
    this.setTabId = options.setTabId ?? ((id: string) => writeStorage("sessionStorage", "gu_gameplay_tab_id", id));
    if (options.onConflictRotated) {
      this.conflictListeners.add(options.onConflictRotated);
    }
    const factory = options.createBroadcastChannel ?? defaultBroadcastChannel;
    this.channel = factory(TAB_CHANNEL_NAME);
    if (this.channel) {
      this.channel.onmessage = (event: { data: unknown }) => {
        this.handleMessage(event.data);
      };
    }
  }

  public onConflictRotated(listener: (rotatedTabId: string) => void): () => void {
    this.conflictListeners.add(listener);
    return () => this.conflictListeners.delete(listener);
  }

  public ensureTabId(): string {
    let current = this.getTabId();
    if (!current) {
      current = this.generateId();
      this.setTabId(current);
    }
    if (!this.channel) return current;

    this.conflictDetected = false;
    try {
      this.channel.postMessage({
        type: "probe",
        tabId: current,
        instanceId: this.instanceId,
      });
    } catch {
      return current;
    }

    if (this.conflictDetected) {
      const latest = this.getTabId();
      if (latest && latest !== current) {
        current = latest;
      } else {
        current = this.generateId();
        this.setTabId(current);
      }
      this.conflictDetected = false;
    }
    return current;
  }

  public close(): void {
    try {
      this.channel?.close();
    } catch {
      // ignore
    }
    this.channel = null;
  }

  private handleMessage(raw: unknown): void {
    if (!raw || typeof raw !== "object") return;
    const msg = raw as { type?: string; tabId?: string; instanceId?: string };
    if (!msg.tabId || !msg.instanceId || msg.instanceId === this.instanceId) return;
    const myTabId = this.getTabId();
    if (!myTabId || msg.tabId !== myTabId) return;

    if (msg.type === "probe") {
      try {
        this.channel?.postMessage({
          type: "in_use",
          tabId: myTabId,
          instanceId: this.instanceId,
        });
      } catch {
        // ignore
      }
      return;
    }

    if (msg.type === "in_use") {
      this.conflictDetected = true;
      const rotated = this.generateId();
      this.setTabId(rotated);
      for (const listener of this.conflictListeners) {
        try {
          listener(rotated);
        } catch {
          // ignore
        }
      }
    }
  }
}

export const defaultTabCoordinator = new TabIdCoordinator();
