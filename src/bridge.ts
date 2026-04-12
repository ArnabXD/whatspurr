import { spawn, type ChildProcess } from "node:child_process";
import { randomUUID, randomBytes } from "node:crypto";
import { platform, arch } from "node:os";
import { join, dirname } from "node:path";
import { existsSync, mkdirSync, chmodSync } from "node:fs";
import { EventEmitter } from "node:events";
import type {
  Command,
  CommandResponse,
  EventMessage,
  WireMessage,
  WhatsAppConfig,
} from "./types.ts";

const DEFAULT_SESSION_DIR = "./session";
const DEFAULT_BINARY_REPO = "ArnabXD/whatspurr";
const STARTUP_TIMEOUT_MS = 10_000;
const COMMAND_TIMEOUT_MS = 30_000;
const LISTEN_ADDR_RE = /^127\.0\.0\.1:\d+$/;

function getPlatformKey(): { goos: string; goarch: string; ext: string } {
  const os = platform();
  const cpu = arch();

  const goos = os === "win32" ? "windows" : os;
  const goarch = cpu === "x64" ? "amd64" : cpu === "arm64" ? "arm64" : cpu;
  const ext = os === "win32" ? ".exe" : "";

  return { goos, goarch, ext };
}

function getBinaryPath(): string {
  const { ext } = getPlatformKey();
  return join(import.meta.dirname ?? ".", "..", "bin", `bridge${ext}`);
}

function findBinary(configPath?: string): string | null {
  if (configPath) {
    if (existsSync(configPath)) return configPath;
    throw new Error(`Specified binary not found: ${configPath}`);
  }

  const { ext } = getPlatformKey();
  const candidates = [
    join(import.meta.dirname ?? ".", "..", "bin", `bridge${ext}`),
    join("bin", `bridge${ext}`),
  ];
  for (const p of candidates) {
    if (existsSync(p)) return p;
  }
  return null;
}

async function downloadBinary(repo: string, version: string): Promise<string> {
  const { goos, goarch, ext } = getPlatformKey();
  const assetName = `bridge-${goos}-${goarch}${ext}`;

  const baseUrl = version === "latest"
    ? `https://github.com/${repo}/releases/latest/download/${assetName}`
    : `https://github.com/${repo}/releases/download/${version}/${assetName}`;

  const dest = getBinaryPath();
  const destDir = dirname(dest);
  if (!existsSync(destDir)) {
    mkdirSync(destDir, { recursive: true });
  }

  console.log(`Downloading bridge binary from ${repo} (${version})...`);
  console.log(`  ${goos}/${goarch} → ${dest}`);

  const resp = await fetch(baseUrl, { redirect: "follow" });
  if (!resp.ok) {
    throw new Error(
      `Failed to download binary: ${resp.status} ${resp.statusText}\n` +
      `  URL: ${baseUrl}\n` +
      `  Try building locally: bun run scripts/build-go.ts`
    );
  }

  const buffer = await resp.arrayBuffer();
  const tmpDest = dest + ".tmp";
  const fs = await import("node:fs/promises");
  await fs.writeFile(tmpDest, Buffer.from(buffer));
  await fs.rename(tmpDest, dest);

  if (goos !== "windows") {
    chmodSync(dest, 0o755);
  }

  console.log(`  Done (${(buffer.byteLength / 1024 / 1024).toFixed(1)} MB)`);
  return dest;
}

export class Bridge extends EventEmitter {
  private process: ChildProcess | null = null;
  private ws: WebSocket | null = null;
  private pending = new Map<string, {
    resolve: (v: Record<string, unknown>) => void;
    reject: (e: Error) => void;
  }>();
  private listenAddr: string | null = null;
  private authToken: string;
  private config: WhatsAppConfig;
  private _ready = false;

  constructor(config: WhatsAppConfig = {}) {
    super();
    this.config = config;
    this.authToken = randomBytes(32).toString("hex");
  }

  get ready(): boolean {
    return this._ready;
  }

  async start(): Promise<void> {
    let binaryPath = findBinary(this.config.binaryPath);
    if (!binaryPath) {
      const repo = this.config.binaryRepo ?? DEFAULT_BINARY_REPO;
      const version = this.config.binaryVersion ?? "latest";
      binaryPath = await downloadBinary(repo, version);
    }
    const sessionDir = this.config.sessionDir ?? DEFAULT_SESSION_DIR;
    const logLevel = this.config.logLevel ?? "info";

    const dbName = this.config.dbName ?? "whatspurr.db";

    const autoPresence = this.config.autoPresence !== false; // default true

    const args = [
      "--token", this.authToken,
      "--session-dir", sessionDir,
      "--db-name", dbName,
      "--log-level", logLevel,
      ...(autoPresence ? ["--auto-presence"] : []),
    ];

    this.process = spawn(binaryPath, args, {
      stdio: ["ignore", "pipe", "pipe"],
    });

    this.process.on("exit", (code, signal) => {
      this._ready = false;
      this.emit("exit", { code, signal });
    });

    this.process.stderr?.on("data", (chunk: Buffer) => {
      const line = chunk.toString().trim();
      if (line) this.emit("log", line);
    });

    // Wait for Go to signal readiness and report listen address
    await this.waitForReady();

    // Connect WebSocket
    await this.connectWs();
  }

  private waitForReady(): Promise<void> {
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error("Go bridge startup timed out"));
      }, STARTUP_TIMEOUT_MS);

      const onData = (chunk: Buffer) => {
        const line = chunk.toString().trim();
        if (line.startsWith("ready ")) {
          const addr = line.slice(6);
          if (!LISTEN_ADDR_RE.test(addr)) {
            clearTimeout(timeout);
            reject(new Error(`Invalid listen address from bridge: ${addr.slice(0, 40)}`));
            return;
          }
          this.listenAddr = addr;
          clearTimeout(timeout);
          this.process?.stdout?.off("data", onData);
          resolve();
        }
      };

      this.process?.stdout?.on("data", onData);

      this.process?.on("error", (err) => {
        clearTimeout(timeout);
        reject(new Error(`Failed to start Go bridge: ${err.message}`));
      });

      this.process?.on("exit", (code) => {
        clearTimeout(timeout);
        if (!this._ready) {
          reject(new Error(`Go bridge exited during startup with code ${code}`));
        }
      });
    });
  }

  private connectWs(): Promise<void> {
    return new Promise((resolve, reject) => {
      const url = `ws://${this.listenAddr}/?token=${this.authToken}`;
      this.ws = new WebSocket(url);

      this.ws.onopen = () => {
        this._ready = true;
        resolve();
      };

      this.ws.onerror = (ev) => {
        if (!this._ready) {
          reject(new Error(`WebSocket connection failed`));
        }
        this.emit("error", ev);
      };

      this.ws.onclose = () => {
        this._ready = false;
        this.emit("ws_close");
      };

      this.ws.onmessage = (ev) => {
        this.handleMessage(ev.data as string);
      };
    });
  }

  private handleMessage(raw: string): void {
    let msg: WireMessage;
    try {
      msg = JSON.parse(raw);
    } catch {
      this.emit("error", new Error(`Invalid JSON from bridge: ${raw.slice(0, 200)}`));
      return;
    }

    // Event push from Go
    if ("type" in msg && msg.type === "event") {
      const event = msg as EventMessage;
      this.emit("event", { event: event.event, data: event.data });
      return;
    }

    // Command response
    const resp = msg as CommandResponse;
    if (resp.id && this.pending.has(resp.id)) {
      const { resolve, reject } = this.pending.get(resp.id)!;
      this.pending.delete(resp.id);
      if (resp.error) {
        reject(new Error(`[${resp.error.code}] ${resp.error.message}`));
      } else {
        resolve(resp.result ?? {});
      }
    }
  }

  /** Send a command to the Go bridge and await the response. */
  async send(method: string, params: Record<string, unknown> = {}): Promise<Record<string, unknown>> {
    if (!this.ws || !this._ready) {
      throw new Error("Bridge is not connected");
    }

    const id = randomUUID();
    const cmd: Command = { id, method, params };

    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.pending.delete(id);
        reject(new Error(`Command '${method}' timed out after ${COMMAND_TIMEOUT_MS}ms`));
      }, COMMAND_TIMEOUT_MS);

      this.pending.set(id, {
        resolve: (v) => { clearTimeout(timer); resolve(v); },
        reject: (e) => { clearTimeout(timer); reject(e); },
      });
      this.ws!.send(JSON.stringify(cmd));
    });
  }

  async stop(): Promise<void> {
    this._ready = false;

    // Reject all pending commands
    for (const [, { reject }] of this.pending) {
      reject(new Error("Bridge is shutting down"));
    }
    this.pending.clear();

    // Close WebSocket
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }

    // Kill Go process
    if (this.process) {
      this.process.kill("SIGTERM");
      await new Promise<void>((resolve) => {
        this.process?.on("exit", () => resolve());
        setTimeout(resolve, 3000); // force resolve after 3s
      });
      this.process = null;
    }
  }
}