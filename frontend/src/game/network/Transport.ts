import type { BufferedWorldState } from "./SnapshotBuffer.ts";
import type {
  BasicAttackPayload,
  CapturePayload,
  CastPayload,
  CompanionStancePayload,
  WorldEvent,
  WorldJoinPayload,
  WorldRejectPayload,
  WorldTransferPayload,
  WorldWelcomePayload,
} from "./Protocol.ts";

/**
 * Transport-agnostic surface consumed by WorldRuntime. Both WorldSocket (v1
 * JSON/WS) and WorldTransport (v2 binary/WebTransport) implement it.
 */
export interface WorldTransportLike {
  connect(join: WorldJoinPayload): Promise<WorldWelcomePayload>;
  isJoined(): boolean;
  getWelcome(): WorldWelcomePayload | null;
  getSnapshot(): BufferedWorldState | null;
  sendInput(move?: { x: number; y: number }): number | null;
  sendMoveTarget(target: { x: number; y: number }): number | null;
  sendCast(cast: Omit<CastPayload, "seq">): number | null;
  sendBasicAttack(attack: Omit<BasicAttackPayload, "seq">): number | null;
  sendCapture(targetId: string, guId?: string): number | null;
  sendCompanionStance(stance: "DEFEND" | "FOCUS" | "FOLLOW", focusTargetId?: string): number | null;
  onEvents(callback: (event: WorldEvent) => void): () => void;
  onSnapshot(callback: (state: BufferedWorldState) => void): () => void;
  onReject(callback: (reject: WorldRejectPayload) => void): () => void;
  onTransfer(callback: (transfer: WorldTransferPayload) => void): () => void;
  onResyncRequired(callback: (reason: string) => void): () => void;
  onError(callback: (error: Error) => void): () => void;
  disconnect(): void;
  getMetrics(): {
    messages: number;
    bytes: number;
    messagesSent: number;
    bytesSent: number;
    snapshots: number;
    lastMessageAt: number;
    lastSnapshotAt: number;
    lastCloseCode: number;
  };
}
