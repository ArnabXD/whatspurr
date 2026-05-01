import { Api } from "./api.ts";
import { Bridge } from "./bridge.ts";
import { Composer } from "./composer.ts";
import { Context } from "./context.ts";
import type { EventData, SessionEventData, WhatsAppConfig } from "./types.ts";

export class WhatsApp extends Composer<Context> {
  readonly api: Api;
  private bridge: Bridge;
  private ownsBridge: boolean;
  private sessionName: string;
  private listenersRegistered = false;
  private boundEventHandler: ((data: SessionEventData) => void) | null = null;
  private boundLogHandler: ((line: string) => void) | null = null;
  private boundExitHandler: ((info: { code: number | null; signal: string | null }) => void) | null = null;
  private boundWsCloseHandler: (() => void) | null = null;
  private started = false;

  constructor(config?: WhatsAppConfig);
  constructor(bridge: Bridge, sessionName: string, downloadDir?: string);
  constructor(configOrBridge?: WhatsAppConfig | Bridge, sessionName?: string, downloadDir?: string) {
    super();
    if (configOrBridge instanceof Bridge) {
      // Managed mode: bridge is shared, session name is required
      this.bridge = configOrBridge;
      this.ownsBridge = false;
      this.sessionName = sessionName ?? "default";
      this.api = new Api(this.bridge, downloadDir, this.sessionName);
    } else {
      // Standalone mode: owns its own bridge
      const config = configOrBridge ?? {};
      this.bridge = new Bridge(config);
      this.ownsBridge = true;
      this.sessionName = config.sessionName ?? "default";
      this.api = new Api(this.bridge, config.downloadDir, this.sessionName);
    }
  }

  /** Start the WhatsApp client (spawns Go bridge, connects WS, connects session) */
  async start(): Promise<void> {
    if (this.started) return;
    this.started = true;

    this.initListeners();

    if (this.ownsBridge) {
      await this.bridge.start();
    }

    // Connect this session
    await this.bridge.send("connect_session", { name: this.sessionName });
  }

  /** Register bridge event listeners. Safe to call before connect so no events are missed. */
  initListeners(): void {
    if (this.listenersRegistered) return;
    this.listenersRegistered = true;

    // Only register bridge-level listeners in standalone mode
    if (this.ownsBridge) {
      this.boundLogHandler = (line: string) => this.onLog(line);
      this.bridge.on("log", this.boundLogHandler);

      this.boundExitHandler = (info: { code: number | null; signal: string | null }) => this.onBridgeExit(info);
      this.bridge.on("exit", this.boundExitHandler);

      this.boundWsCloseHandler = () => this.onWsClose();
      this.bridge.on("ws_close", this.boundWsCloseHandler);
    }

    // Handle WhatsApp events from the bridge (filtered by session)
    this.boundEventHandler = (eventData: SessionEventData) => {
      if (eventData.session === this.sessionName) {
        this.handleEvent({ event: eventData.event, data: eventData.data } as EventData);
      }
    };
    this.bridge.on("event", this.boundEventHandler);
  }

  /** Remove bridge event listeners. */
  private removeListeners(): void {
    if (!this.listenersRegistered) return;
    this.listenersRegistered = false;

    if (this.boundEventHandler) {
      this.bridge.off("event", this.boundEventHandler);
      this.boundEventHandler = null;
    }
    if (this.boundLogHandler) {
      this.bridge.off("log", this.boundLogHandler);
      this.boundLogHandler = null;
    }
    if (this.boundExitHandler) {
      this.bridge.off("exit", this.boundExitHandler);
      this.boundExitHandler = null;
    }
    if (this.boundWsCloseHandler) {
      this.bridge.off("ws_close", this.boundWsCloseHandler);
      this.boundWsCloseHandler = null;
    }
  }

  /** Stop the WhatsApp client gracefully */
  async stop(): Promise<void> {
    this.removeListeners();
    this.started = false;

    // Disconnect session (preserves auth data)
    try {
      await this.bridge.send("disconnect_session", { name: this.sessionName });
    } catch {
      // Bridge may already be down
    }

    if (this.ownsBridge) {
      await this.bridge.stop();
    }
  }

  private async handleEvent(eventData: EventData): Promise<void> {
    if (!this.started) return;
    const ctx = new Context(eventData, this.api);
    const mw = this.middleware();

    try {
      await mw(ctx, async () => {});
    } catch (err) {
      if (!this.started) return; // bridge shut down mid-handler; error is expected
      this.onError(err instanceof Error ? err : new Error(String(err)), ctx);
    }
  }

  /** Override to customize logging. Default: console.log */
  protected onLog(line: string): void {
    console.log(`[bridge] ${line}`);
  }

  /** Override to handle WebSocket disconnects (auto-reconnect is built-in; override for custom logic) */
  protected onWsClose(): void {}

  /** Override to handle bridge process exit */
  protected onBridgeExit(info: { code: number | null; signal: string | null }): void {
    console.error(`[bridge] process exited (code=${info.code}, signal=${info.signal})`);
  }

  /** Override to handle unhandled middleware errors */
  protected onError(err: Error, _ctx: Context): void {
    console.error("[whatsapp] unhandled error:", err);
  }
}
