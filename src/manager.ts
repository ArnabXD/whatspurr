import { Bridge } from "./bridge.ts";
import type { SessionInfo, WhatsAppConfig } from "./types.ts";
import { WhatsApp } from "./whatsapp.ts";

export class WhatsAppManager {
  private bridge: Bridge;
  private clients = new Map<string, WhatsApp>();
  private config: WhatsAppConfig;
  private started = false;

  constructor(config: WhatsAppConfig = {}) {
    this.config = config;
    this.bridge = new Bridge(config);
  }

  /** Start the Go bridge process. No sessions are connected yet. */
  async start(): Promise<void> {
    if (this.started) return;
    // Recreate bridge if previously stopped
    if (!this.bridge.ready) {
      this.bridge = new Bridge(this.config);
    }
    await this.bridge.start();
    this.started = true;
  }

  /**
   * Connect a session by name. Returns a WhatsApp instance with its own middleware stack.
   *
   * The returned instance is ready for registering handlers (`on`, `hears`, `command`).
   * Call `wa.start()` after registering handlers to actually connect to WhatsApp.
   * This ensures no events are missed between connect and handler registration.
   *
   * @example
   * ```ts
   * const wa = await mgr.connect("bot1");
   * wa.on("qr", ctx => console.log(ctx.qr.code));
   * wa.on("message:text", ctx => ctx.reply("echo"));
   * await wa.start(); // now connects
   * ```
   */
  async connect(name: string): Promise<WhatsApp> {
    if (!this.started) {
      await this.start();
    }

    const existing = this.clients.get(name);
    if (existing) {
      return existing;
    }

    const wa = new WhatsApp(this.bridge, name, this.config.downloadDir);
    // Register bridge listeners now so events are captured as soon as start() fires
    wa.initListeners();
    this.clients.set(name, wa);

    return wa;
  }

  /** Disconnect a session (preserves auth data in DB). Can reconnect later. */
  async disconnect(name: string): Promise<void> {
    const wa = this.clients.get(name);
    if (!wa) return;

    await wa.stop();
    this.clients.delete(name);
  }

  /** Destroy a session — logout from WhatsApp and delete auth data from DB. */
  async destroy(name: string): Promise<void> {
    const wa = this.clients.get(name);
    if (wa) {
      await wa.stop();
      this.clients.delete(name);
    }
    await this.bridge.send("destroy_session", { name });
  }

  /** List all sessions in the database with their connection status. */
  async list(): Promise<SessionInfo[]> {
    const result = await this.bridge.send("list_sessions", {});
    return (result.sessions as SessionInfo[]) ?? [];
  }

  /** Stop all sessions and kill the Go bridge process. */
  async stop(): Promise<void> {
    for (const wa of this.clients.values()) {
      await wa.stop();
    }
    this.clients.clear();
    await this.bridge.stop();
    this.started = false;
  }
}
