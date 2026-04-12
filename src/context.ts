import type { Api } from "./api.ts";
import type {
  Message,
  JID,
  SendResult,
  MediaSendOptions,
  EventData,
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

  /** React to the current message with an emoji */
  async react(emoji: string): Promise<SendResult> {
    const chat = this.chat;
    const msgId = this.message?.id;
    if (!chat || !msgId) throw new Error("No message to react to");
    return this.api.sendReaction(chat, msgId, emoji);
  }
}