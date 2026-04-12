// ── JID ──────────────────────────────────────────────────────────────────────
/** WhatsApp JID string, e.g. "1234567890@s.whatsapp.net" or "120363xxx@g.us" */
export type JID = string;

// ── Config ───────────────────────────────────────────────────────────────────
export interface WhatsAppConfig {
  /** Directory for session/auth data. Default: "./session" */
  sessionDir?: string;
  /** SQLite database filename. Default: "whatspurr.db" */
  dbName?: string;
  /** Path to Go bridge binary. Default: auto-detect in bin/ */
  binaryPath?: string;
  /** Log level for Go sidecar: "debug" | "info" | "warn" | "error" */
  logLevel?: "debug" | "info" | "warn" | "error";
  /** Automatically send "available" presence on connect. Default: true */
  autoPresence?: boolean;
}

// ── Messages ─────────────────────────────────────────────────────────────────
export type MessageType =
  | "text"
  | "image"
  | "video"
  | "audio"
  | "document"
  | "sticker"
  | "contact"
  | "location";

export interface MessageInfo {
  id: string;
  from: JID;
  chat: JID;
  pushName: string;
  timestamp: number;
  isGroup: boolean;
}

export interface TextMessage extends MessageInfo {
  type: "text";
  text: string;
}

export interface MediaMessage extends MessageInfo {
  type: "image" | "video" | "audio" | "document" | "sticker";
  caption?: string;
  mimetype: string;
  filename?: string;
  /** Opaque ref used to download media via bridge */
  mediaRef: string;
}

export interface ContactMessage extends MessageInfo {
  type: "contact";
  displayName: string;
  vcard: string;
}

export interface LocationMessage extends MessageInfo {
  type: "location";
  latitude: number;
  longitude: number;
  name?: string;
  address?: string;
}

export type Message = TextMessage | MediaMessage | ContactMessage | LocationMessage;

// ── Reactions ────────────────────────────────────────────────────────────────
export interface ReactionEvent {
  from: JID;
  chat: JID;
  messageId: string;
  emoji: string;
  timestamp: number;
}

// ── Receipts ─────────────────────────────────────────────────────────────────
export type ReceiptType = "delivered" | "read" | "played";

export interface ReceiptEvent {
  from: JID;
  chat: JID;
  messageIds: string[];
  type: ReceiptType;
  timestamp: number;
}

// ── Presence ─────────────────────────────────────────────────────────────────
export type PresenceType = "available" | "unavailable" | "composing" | "recording";

export interface PresenceEvent {
  from: JID;
  chat: JID;
  type: PresenceType;
  lastSeen?: number;
}

// ── Groups ───────────────────────────────────────────────────────────────────
export interface GroupInfo {
  jid: JID;
  name: string;
  topic: string;
  participants: GroupParticipant[];
  createdAt: number;
}

export interface GroupParticipant {
  jid: JID;
  isAdmin: boolean;
  isSuperAdmin: boolean;
}

export interface GroupJoinEvent {
  chat: JID;
  participants: JID[];
  addedBy?: JID;
}

export interface GroupLeaveEvent {
  chat: JID;
  participants: JID[];
  removedBy?: JID;
}

export interface GroupUpdateEvent {
  chat: JID;
  field: "name" | "topic" | "photo" | "locked" | "announce";
  value: string;
  updatedBy: JID;
}

// ── Events ───────────────────────────────────────────────────────────────────
export type EventType =
  | "qr"
  | "connected"
  | "disconnected"
  | "message"
  | "message_reaction"
  | "receipt"
  | "presence"
  | "group_join"
  | "group_leave"
  | "group_update";

export interface QrEvent {
  code: string;
}

export interface ConnectedEvent {
  jid: JID;
}

export interface DisconnectedEvent {
  reason: string;
}

export type EventData =
  | { event: "qr"; data: QrEvent }
  | { event: "connected"; data: ConnectedEvent }
  | { event: "disconnected"; data: DisconnectedEvent }
  | { event: "message"; data: Message }
  | { event: "message_reaction"; data: ReactionEvent }
  | { event: "receipt"; data: ReceiptEvent }
  | { event: "presence"; data: PresenceEvent }
  | { event: "group_join"; data: GroupJoinEvent }
  | { event: "group_leave"; data: GroupLeaveEvent }
  | { event: "group_update"; data: GroupUpdateEvent };

// ── Protocol (WS wire format) ────────────────────────────────────────────────
export interface Command {
  id: string;
  method: string;
  params: Record<string, unknown>;
}

export interface CommandResponse {
  id: string;
  result?: Record<string, unknown>;
  error?: { code: number; message: string };
}

export interface EventMessage {
  type: "event";
  event: EventType;
  data: unknown;
}

export type WireMessage = CommandResponse | EventMessage;

// ── Middleware ────────────────────────────────────────────────────────────────
export type NextFn = () => Promise<void>;
export type MiddlewareFn<C> = (ctx: C, next: NextFn) => unknown | Promise<unknown>;

// ── API result types ─────────────────────────────────────────────────────────
export interface SendResult {
  messageId: string;
}

export interface MediaSendOptions {
  caption?: string;
  filename?: string;
  mimetype?: string;
}