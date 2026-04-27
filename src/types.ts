// ── JID ──────────────────────────────────────────────────────────────────────
/** WhatsApp JID string, e.g. "1234567890@s.whatsapp.net" or "120363xxx@g.us" */
export type JID = string;

// ── Config ───────────────────────────────────────────────────────────────────
export interface WhatsAppConfig {
  /** Session name for identifying this device. Default: "default" */
  sessionName?: string;
  /** Directory for session/auth data. Default: "./session" */
  sessionDir?: string;
  /** SQLite database filename. Default: "whatspurr.db" */
  dbName?: string;
  /** Path to Go bridge binary. Default: auto-detect in bin/, download if missing */
  binaryPath?: string;
  /** Log level for Go sidecar: "debug" | "info" | "warn" | "error" */
  logLevel?: "debug" | "info" | "warn" | "error";
  /** Automatically send "available" presence on connect. Default: true */
  autoPresence?: boolean;
  /** Subscribe to outgoing messages (sent by us). Default: false */
  subscribeOutgoing?: boolean;
  /** GitHub owner/repo for binary downloads. Default: "ArnabXD/whatspurr" */
  binaryRepo?: string;
  /** Version tag to download. Default: "latest" */
  binaryVersion?: string;
  /** Directory for downloaded media files. Default: "./downloads" */
  downloadDir?: string;
}

// ── Messages ─────────────────────────────────────────────────────────────────
export type MessageType = "text" | "image" | "video" | "audio" | "document" | "sticker" | "contact" | "location";

// ── Content shapes (shared by messages and quoted messages) ─────────────────
export interface TextContent {
  type: "text";
  text: string;
}

export interface MediaContent {
  type: "image" | "video" | "audio" | "document" | "sticker";
  caption?: string;
  mimetype: string;
  filename?: string;
}

export interface ContactContent {
  type: "contact";
  displayName: string;
}

export interface LocationContent {
  type: "location";
  latitude: number;
  longitude: number;
}

// ── Quoted messages ─────────────────────────────────────────────────────────
export interface QuotedMessageBase {
  messageId: string;
  sender: JID;
}

export interface QuotedTextMessage extends QuotedMessageBase, TextContent {}
export interface QuotedMediaMessage extends QuotedMessageBase, MediaContent {}
export interface QuotedContactMessage extends QuotedMessageBase, ContactContent {}
export interface QuotedLocationMessage extends QuotedMessageBase, LocationContent {}

export interface QuotedUnknownMessage extends QuotedMessageBase {
  type?: undefined;
}

export type QuotedMessage =
  | QuotedTextMessage
  | QuotedMediaMessage
  | QuotedContactMessage
  | QuotedLocationMessage
  | QuotedUnknownMessage;

// ── Messages ────────────────────────────────────────────────────────────────
export interface MessageInfo {
  id: string;
  from: JID;
  chat: JID;
  pushName: string;
  timestamp: number;
  isGroup: boolean;
  isFromMe: boolean;
  /** Present when this message is a reply to another message */
  quotedMessage?: QuotedMessage;
}

export interface TextMessage extends MessageInfo, TextContent {}

export interface MediaMessage extends MessageInfo, MediaContent {
  /** Opaque ref used to download media via bridge */
  mediaRef: string;
  /** Whether this is a view-once media message */
  viewOnce?: boolean;
  /** Media width in pixels (images and videos) */
  width?: number;
  /** Media height in pixels (images and videos) */
  height?: number;
}

export interface ContactMessage extends MessageInfo, ContactContent {
  vcard: string;
}

export interface LocationMessage extends MessageInfo, LocationContent {
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
  session?: string;
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
  session?: string;
  event: EventType;
  data: unknown;
}

export type WireMessage = CommandResponse | EventMessage;

// ── Session Management ──────────────────────────────────────────────────────
export interface SessionEventData {
  session: string;
  event: EventType;
  data: unknown;
}

export interface SessionInfo {
  name: string;
  jid: string;
  connected: boolean;
}

// ── Context Narrowing ────────────────────────────────────────────────────────

type MessageNarrow = {
  readonly message: Message;
  readonly from: JID;
  readonly chat: JID;
};

type TextMessageNarrow = MessageNarrow & {
  readonly message: TextMessage;
  readonly text: string;
};

type MediaMessageNarrow = MessageNarrow & {
  readonly message: MediaMessage;
};

type ContactMessageNarrow = MessageNarrow & {
  readonly message: ContactMessage;
};

type LocationMessageNarrow = MessageNarrow & {
  readonly message: LocationMessage;
};

type FilterNarrowMap = {
  message: MessageNarrow;
  "message:text": TextMessageNarrow;
  "message:image": MediaMessageNarrow;
  "message:video": MediaMessageNarrow;
  "message:audio": MediaMessageNarrow;
  "message:document": MediaMessageNarrow;
  "message:sticker": MediaMessageNarrow;
  "message:contact": ContactMessageNarrow;
  "message:location": LocationMessageNarrow;
  message_reaction: { readonly reaction: ReactionEvent };
  receipt: { readonly receipt: ReceiptEvent };
  presence: { readonly presence: PresenceEvent };
  qr: { readonly qr: QrEvent };
  connected: { readonly connected: ConnectedEvent };
  disconnected: { readonly disconnected: DisconnectedEvent };
  group_join: { readonly groupJoin: GroupJoinEvent };
  group_leave: { readonly groupLeave: GroupLeaveEvent };
  group_update: { readonly groupUpdate: GroupUpdateEvent };
};

export type FilterQuery = keyof FilterNarrowMap;
export type NarrowContext<C, Q extends FilterQuery> = C & FilterNarrowMap[Q];

// ── Middleware ────────────────────────────────────────────────────────────────
export type NextFn = () => Promise<void>;
export type MiddlewareFn<C> = (ctx: C, next: NextFn) => unknown | Promise<unknown>;

// ── API result types ─────────────────────────────────────────────────────────
export interface SendResult {
  messageId: string;
}

export interface DownloadResult {
  /** Absolute path to the downloaded file */
  path: string;
  /** File size in bytes */
  size: number;
}

export interface QuoteOptions {
  /** Message ID to quote (creates a quoted reply) */
  quotedMessageId?: string;
  /** Sender JID of the quoted message (required for group quotes) */
  quotedSender?: JID;
}

export interface SendOptions extends QuoteOptions {}

export interface MediaSendOptions extends QuoteOptions {
  caption?: string;
  filename?: string;
  mimetype?: string;
  /** Send as a view-once media message (image/video only) */
  viewOnce?: boolean;
  /** Media width in pixels (image/video only) */
  width?: number;
  /** Media height in pixels (image/video only) */
  height?: number;
}
