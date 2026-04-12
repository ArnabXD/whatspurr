import type { Api } from "./api.ts";
import type {
  ConnectedEvent,
  DisconnectedEvent,
  EventData,
  GroupJoinEvent,
  GroupLeaveEvent,
  GroupUpdateEvent,
  JID,
  MediaSendOptions,
  Message,
  PresenceEvent,
  QrEvent,
  ReactionEvent,
  ReceiptEvent,
  SendResult,
} from "./types.ts";

export class Context {
  /** The raw event data */
  readonly eventData: EventData;

  /** The API instance for sending messages */
  readonly api: Api;

  /** Set by `hears()` — the regex match result */
  match?: RegExpMatchArray;

  /** Set by `command()` — the text after the command */
  commandArgs?: string;

  constructor(eventData: EventData, api: Api) {
    this.eventData = eventData;
    this.api = api;
  }

  /** The event type (e.g. "message", "qr", "connected") */
  get eventType(): EventData["event"] {
    return this.eventData.event;
  }

  /** The message object (only on "message" events) */
  get message(): Message | undefined {
    if (this.eventData.event === "message") {
      return this.eventData.data as Message;
    }
    return undefined;
  }

  /** The text content of the message, if any */
  get text(): string | undefined {
    const msg = this.message;
    if (msg?.type === "text") return msg.text;
    return undefined;
  }

  /** Sender JID */
  get from(): JID | undefined {
    const msg = this.message;
    return msg?.from;
  }

  /** Chat JID (same as from for DMs, group JID for groups) */
  get chat(): JID | undefined {
    const msg = this.message;
    return msg?.chat;
  }

  /** Whether this message was sent in a group */
  get isGroup(): boolean {
    return this.message?.isGroup ?? false;
  }

  /** Whether this message was sent by us (outgoing) */
  get isFromMe(): boolean {
    return this.message?.isFromMe ?? false;
  }

  /** QR event data (only on "qr" events) */
  get qr(): QrEvent | undefined {
    return this.eventData.event === "qr" ? this.eventData.data : undefined;
  }

  /** Connected event data (only on "connected" events) */
  get connected(): ConnectedEvent | undefined {
    return this.eventData.event === "connected" ? this.eventData.data : undefined;
  }

  /** Disconnected event data (only on "disconnected" events) */
  get disconnected(): DisconnectedEvent | undefined {
    return this.eventData.event === "disconnected" ? this.eventData.data : undefined;
  }

  /** Reaction event data (only on "message_reaction" events) */
  get reaction(): ReactionEvent | undefined {
    return this.eventData.event === "message_reaction" ? this.eventData.data : undefined;
  }

  /** Receipt event data (only on "receipt" events) */
  get receipt(): ReceiptEvent | undefined {
    return this.eventData.event === "receipt" ? this.eventData.data : undefined;
  }

  /** Presence event data (only on "presence" events) */
  get presence(): PresenceEvent | undefined {
    return this.eventData.event === "presence" ? this.eventData.data : undefined;
  }

  /** Group join event data (only on "group_join" events) */
  get groupJoin(): GroupJoinEvent | undefined {
    return this.eventData.event === "group_join" ? this.eventData.data : undefined;
  }

  /** Group leave event data (only on "group_leave" events) */
  get groupLeave(): GroupLeaveEvent | undefined {
    return this.eventData.event === "group_leave" ? this.eventData.data : undefined;
  }

  /** Group update event data (only on "group_update" events) */
  get groupUpdate(): GroupUpdateEvent | undefined {
    return this.eventData.event === "group_update" ? this.eventData.data : undefined;
  }

  /** Reply with a text message to the current chat */
  async reply(text: string): Promise<SendResult> {
    const chat = this.chat;
    if (!chat) throw new Error("No chat to reply to");
    return this.api.sendMessage(chat, text);
  }

  /** Reply with an image to the current chat */
  async replyWithImage(data: Buffer | Uint8Array, options: MediaSendOptions = {}): Promise<SendResult> {
    const chat = this.chat;
    if (!chat) throw new Error("No chat to reply to");
    return this.api.sendImage(chat, data, options);
  }

  /** Reply with a video to the current chat */
  async replyWithVideo(data: Buffer | Uint8Array, options: MediaSendOptions = {}): Promise<SendResult> {
    const chat = this.chat;
    if (!chat) throw new Error("No chat to reply to");
    return this.api.sendVideo(chat, data, options);
  }

  /** Reply with audio to the current chat */
  async replyWithAudio(data: Buffer | Uint8Array, options: MediaSendOptions = {}): Promise<SendResult> {
    const chat = this.chat;
    if (!chat) throw new Error("No chat to reply to");
    return this.api.sendAudio(chat, data, options);
  }

  /** Reply with a document to the current chat */
  async replyWithDocument(data: Buffer | Uint8Array, options: MediaSendOptions = {}): Promise<SendResult> {
    const chat = this.chat;
    if (!chat) throw new Error("No chat to reply to");
    return this.api.sendDocument(chat, data, options);
  }

  /** Send a "typing" indicator to the current chat */
  async sendTyping(): Promise<void> {
    const chat = this.chat;
    if (!chat) throw new Error("No chat to send typing to");
    return this.api.sendChatPresence(chat, "composing");
  }

  /** Send a "recording audio" indicator to the current chat */
  async sendRecording(): Promise<void> {
    const chat = this.chat;
    if (!chat) throw new Error("No chat to send recording to");
    return this.api.sendChatPresence(chat, "composing", "audio");
  }

  /** Clear the typing/recording indicator in the current chat */
  async sendPaused(): Promise<void> {
    const chat = this.chat;
    if (!chat) throw new Error("No chat to send paused to");
    return this.api.sendChatPresence(chat, "paused");
  }

  /** Mark the current message as read */
  async markRead(): Promise<void> {
    const chat = this.chat;
    const from = this.from;
    const msgId = this.message?.id;
    if (!chat || !from || !msgId) throw new Error("No message to mark as read");
    return this.api.markRead(chat, from, [msgId]);
  }

  /** React to the current message with an emoji */
  async react(emoji: string): Promise<SendResult> {
    const chat = this.chat;
    const msgId = this.message?.id;
    if (!chat || !msgId) throw new Error("No message to react to");
    return this.api.sendReaction(chat, msgId, emoji);
  }
}
