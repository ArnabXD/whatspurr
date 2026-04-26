import { randomUUID } from "node:crypto";
import { resolve } from "node:path";
import type { Bridge } from "./bridge.ts";
import type {
  DownloadResult,
  GroupInfo,
  JID,
  MediaSendOptions,
  QuoteOptions,
  SendOptions,
  SendResult,
} from "./types.ts";

const DEFAULT_DOWNLOAD_DIR = "./downloads";

const MIME_EXTENSIONS: Record<string, string> = {
  "image/jpeg": ".jpg",
  "image/png": ".png",
  "image/webp": ".webp",
  "video/mp4": ".mp4",
  "audio/ogg": ".ogg",
  "audio/mpeg": ".mp3",
  "application/pdf": ".pdf",
};

/** Fallback: derive extension from mimetype subtype (e.g. "audio/mp4" → ".mp4") */
function mimeToExt(mimetype?: string): string {
  if (!mimetype) return "";
  const sub = mimetype.split("/")[1];
  if (!sub) return "";
  return `.${sub.split(";")[0]}`;
}

export class Api {
  private downloadDir: string;
  private sessionName?: string;

  constructor(
    private bridge: Bridge,
    downloadDir?: string,
    sessionName?: string,
  ) {
    this.downloadDir = downloadDir ?? DEFAULT_DOWNLOAD_DIR;
    this.sessionName = sessionName;
  }

  private send(method: string, params: Record<string, unknown> = {}): Promise<Record<string, unknown>> {
    return this.bridge.send(method, params, this.sessionName);
  }

  private quoteParams(options: QuoteOptions) {
    if (!options.quotedMessageId) return {};
    return {
      quotedId: options.quotedMessageId,
      quotedSender: options.quotedSender ?? "",
    };
  }

  async sendMessage(to: JID, text: string, options: SendOptions = {}): Promise<SendResult> {
    const result = await this.send("send_message", { to, text, ...this.quoteParams(options) });
    return { messageId: result.messageId as string };
  }

  async sendImage(to: JID, data: Buffer | Uint8Array, options: MediaSendOptions = {}): Promise<SendResult> {
    const result = await this.send("send_image", {
      to,
      data: Buffer.from(data).toString("base64"),
      mimetype: options.mimetype ?? "image/jpeg",
      caption: options.caption,
      viewOnce: options.viewOnce,
      width: options.width,
      height: options.height,
      ...this.quoteParams(options),
    });
    return { messageId: result.messageId as string };
  }

  async sendVideo(to: JID, data: Buffer | Uint8Array, options: MediaSendOptions = {}): Promise<SendResult> {
    const result = await this.send("send_video", {
      to,
      data: Buffer.from(data).toString("base64"),
      mimetype: options.mimetype ?? "video/mp4",
      caption: options.caption,
      viewOnce: options.viewOnce,
      width: options.width,
      height: options.height,
      ...this.quoteParams(options),
    });
    return { messageId: result.messageId as string };
  }

  async sendAudio(to: JID, data: Buffer | Uint8Array, options: MediaSendOptions = {}): Promise<SendResult> {
    const result = await this.send("send_audio", {
      to,
      data: Buffer.from(data).toString("base64"),
      mimetype: options.mimetype ?? "audio/ogg",
      ...this.quoteParams(options),
    });
    return { messageId: result.messageId as string };
  }

  async sendDocument(to: JID, data: Buffer | Uint8Array, options: MediaSendOptions = {}): Promise<SendResult> {
    const result = await this.send("send_document", {
      to,
      data: Buffer.from(data).toString("base64"),
      mimetype: options.mimetype ?? "application/octet-stream",
      caption: options.caption,
      filename: options.filename,
      ...this.quoteParams(options),
    });
    return { messageId: result.messageId as string };
  }

  async sendReaction(to: JID, messageId: string, emoji: string): Promise<SendResult> {
    const result = await this.send("send_reaction", { to, messageId, emoji });
    return { messageId: result.messageId as string };
  }

  async downloadMedia(mediaRef: string, path?: string, mimetype?: string): Promise<DownloadResult> {
    const ext = MIME_EXTENSIONS[mimetype ?? ""] ?? mimeToExt(mimetype);
    const destPath = resolve(path ?? `${this.downloadDir}/${randomUUID()}${ext}`);
    const result = await this.send("download_media", { mediaRef, path: destPath });
    return { path: result.path as string, size: result.size as number };
  }

  async getGroupInfo(jid: JID): Promise<GroupInfo> {
    const result = await this.send("get_group_info", { jid });
    return result as unknown as GroupInfo;
  }

  async sendChatPresence(to: JID, state: "composing" | "paused", media?: "text" | "audio"): Promise<void> {
    await this.send("send_chat_presence", { to, state, media });
  }

  async markRead(chat: JID, sender: JID, messageIds: string[]): Promise<void> {
    await this.send("mark_read", { chat, sender, messageIds });
  }

  async setPresence(type: "available" | "unavailable"): Promise<void> {
    await this.send("set_presence", { type });
  }
}
