/** Filter strings for use with `wa.on()` */
export const filters = {
  /** Any message */
  message: "message",
  /** Text messages only */
  text: "message:text",
  /** Image messages */
  image: "message:image",
  /** Video messages */
  video: "message:video",
  /** Audio messages */
  audio: "message:audio",
  /** Document messages */
  document: "message:document",
  /** Sticker messages */
  sticker: "message:sticker",
  /** Contact messages */
  contact: "message:contact",
  /** Location messages */
  location: "message:location",
  /** Reaction events */
  reaction: "message_reaction",
  /** QR code for auth */
  qr: "qr",
  /** Connected to WhatsApp */
  connected: "connected",
  /** Disconnected from WhatsApp */
  disconnected: "disconnected",
  /** Delivery/read receipts */
  receipt: "receipt",
  /** Presence updates */
  presence: "presence",
  /** Group member joined */
  groupJoin: "group_join",
  /** Group member left */
  groupLeave: "group_leave",
  /** Group info updated */
  groupUpdate: "group_update",
} as const;
