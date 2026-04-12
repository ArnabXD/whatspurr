// Main class
export { WhatsApp } from "./whatsapp.ts";

// Building blocks
export { Composer } from "./composer.ts";
export { Context } from "./context.ts";
export { Api } from "./api.ts";
export { Bridge } from "./bridge.ts";

// Filters
export { filters } from "./filters.ts";

// Types
export type {
  // Config
  WhatsAppConfig,
  // JID
  JID,
  // Messages
  MessageType,
  MessageInfo,
  TextMessage,
  MediaMessage,
  ContactMessage,
  LocationMessage,
  Message,
  // Events
  EventType,
  EventData,
  QrEvent,
  ConnectedEvent,
  DisconnectedEvent,
  ReactionEvent,
  ReceiptEvent,
  PresenceEvent,
  GroupInfo,
  GroupParticipant,
  GroupJoinEvent,
  GroupLeaveEvent,
  GroupUpdateEvent,
  // API
  SendResult,
  MediaSendOptions,
  // Middleware
  MiddlewareFn,
  NextFn,
} from "./types.ts";