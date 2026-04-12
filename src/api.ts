import type { Bridge } from "./bridge.ts";
import type { GroupInfo, JID, MediaSendOptions, SendResult } from "./types.ts";

export class Api {
  constructor(private bridge: Bridge) {}

  async sendMessage(to: JID, text: string): Promise<SendResult> {
    const result = await this.bridge.send("send_message", { to, text });
    return { messageId: result.messageId as string };
  }

  async sendImage(to: JID, data: Buffer | Uint8Array, options: MediaSendOptions = {}): Promise<SendResult> {
    const result = await this.bridge.send("send_image", {
      to,
      data: Buffer.from(data).toString("base64"),
      mimetype: options.mimetype ?? "image/jpeg",
      caption: options.caption,
    });
    return { messageId: result.messageId as string };
  }

  async sendVideo(to: JID, data: Buffer | Uint8Array, options: MediaSendOptions = {}): Promise<SendResult> {
    const result = await this.bridge.send("send_video", {
      to,
      data: Buffer.from(data).toString("base64"),
      mimetype: options.mimetype ?? "video/mp4",
      caption: options.caption,
    });
    return { messageId: result.messageId as string };
  }

  async sendAudio(to: JID, data: Buffer | Uint8Array, options: MediaSendOptions = {}): Promise<SendResult> {
    const result = await this.bridge.send("send_audio", {
      to,
      data: Buffer.from(data).toString("base64"),
      mimetype: options.mimetype ?? "audio/ogg",
    });
    return { messageId: result.messageId as string };
  }

  async sendDocument(to: JID, data: Buffer | Uint8Array, options: MediaSendOptions = {}): Promise<SendResult> {
    const result = await this.bridge.send("send_document", {
      to,
      data: Buffer.from(data).toString("base64"),
      mimetype: options.mimetype ?? "application/octet-stream",
      caption: options.caption,
      filename: options.filename,
    });
    return { messageId: result.messageId as string };
  }

  async sendReaction(to: JID, messageId: string, emoji: string): Promise<SendResult> {
    const result = await this.bridge.send("send_reaction", { to, messageId, emoji });
    return { messageId: result.messageId as string };
  }

  async getGroupInfo(jid: JID): Promise<GroupInfo> {
    const result = await this.bridge.send("get_group_info", { jid });
    return result as unknown as GroupInfo;
  }

  async sendChatPresence(to: JID, state: "composing" | "paused", media?: "text" | "audio"): Promise<void> {
    await this.bridge.send("send_chat_presence", { to, state, media });
  }

  async markRead(chat: JID, sender: JID, messageIds: string[]): Promise<void> {
    await this.bridge.send("mark_read", { chat, sender, messageIds });
  }

  async setPresence(type: "available" | "unavailable"): Promise<void> {
    await this.bridge.send("set_presence", { type });
  }
}
