import { Api } from "./api.ts";
import { Bridge } from "./bridge.ts";
import { Composer } from "./composer.ts";
import { Context } from "./context.ts";
import type { EventData, WhatsAppConfig } from "./types.ts";

export class WhatsApp extends Composer<Context> {
  readonly api: Api;
  private bridge: Bridge;

  constructor(config: WhatsAppConfig = {}) {
    super();
    this.bridge = new Bridge(config);
    this.api = new Api(this.bridge, config.downloadDir);
  }

  /** Start the WhatsApp client (spawns Go bridge, connects WS) */
  async start(): Promise<void> {
    // Forward bridge-level events
    this.bridge.on("log", (line: string) => this.onLog(line));
    this.bridge.on("exit", (info: { code: number | null; signal: string | null }) => {
      this.onBridgeExit(info);
    });

    // Handle WhatsApp events from the bridge
    this.bridge.on("event", (eventData: EventData) => {
      this.handleEvent(eventData);
    });

    await this.bridge.start();
  }

  /** Stop the WhatsApp client gracefully */
  async stop(): Promise<void> {
    await this.bridge.stop();
  }

  private async handleEvent(eventData: EventData): Promise<void> {
    const ctx = new Context(eventData, this.api);
    const mw = this.middleware();

    try {
      await mw(ctx, async () => {});
    } catch (err) {
      this.onError(err instanceof Error ? err : new Error(String(err)), ctx);
    }
  }

  /** Override to customize logging. Default: console.log */
  protected onLog(line: string): void {
    console.log(`[bridge] ${line}`);
  }

  /** Override to handle bridge process exit */
  protected onBridgeExit(info: { code: number | null; signal: string | null }): void {
    console.error(`[bridge] process exited (code=${info.code}, signal=${info.signal})`);
  }

  /** Override to handle unhandled middleware errors */
  protected onError(err: Error, _ctx: Context): void {
    console.error("[whatsapp] unhandled error:", err);
  }
}
