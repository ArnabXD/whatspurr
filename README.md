# whatspurr

A grammY-style TypeScript library for WhatsApp, powered by [whatsmeow](https://github.com/tulir/whatsmeow) (Go) via a WebSocket sidecar.

## Install

```bash
bun add @arnabxd/whatspurr
```

The Go bridge binary is automatically downloaded on first `wa.start()`. To build from source instead:

```bash
bun run build:go
```

## Example

```ts
import { WhatsApp } from "@arnabxd/whatspurr";
import { renderUnicodeCompact } from "uqr";

const wa = new WhatsApp({
  sessionDir: "./session",
  logLevel: "info",
});

// QR code event — render for scanning
wa.on("qr", (ctx) => {
  const code = (ctx.eventData.data as { code: string }).code;
  console.log("\nScan this QR code in WhatsApp:\n");
  console.log(renderUnicodeCompact(code));
});

// Connected
wa.on("connected", (ctx) => {
  const jid = (ctx.eventData.data as { jid: string }).jid;
  console.log(`Connected as ${jid}`);
});

// Disconnected
wa.on("disconnected", (ctx) => {
  const reason = (ctx.eventData.data as { reason: string }).reason;
  console.log(`Disconnected: ${reason}`);
});

// Echo bot: reply to text messages
wa.on("message:text", async (ctx) => {
  console.log(`${ctx.from}: ${ctx.text}`);
  await ctx.reply(`Echo: ${ctx.text}`);
});

// Start (downloads the Go bridge binary on first run)
await wa.start();

// Graceful shutdown
process.on("SIGINT", async () => {
  await wa.stop();
  process.exit(0);
});
```

## Configuration

```ts
const wa = new WhatsApp({
  sessionDir: "./session",              // Session/auth data directory (default: "./session")
  dbName: "whatspurr.db",               // SQLite database filename (default: "whatspurr.db")
  logLevel: "info",                     // debug | info | warn | error
  binaryPath: "/path/to/bridge",        // Use a specific binary (skip auto-download)
  binaryRepo: "ArnabXD/whatspurr",      // GitHub owner/repo for binary downloads
  binaryVersion: "v0.1.0",              // Pin a release version (default: "latest")
  autoPresence: true,                   // Send "available" presence on connect (default: true)
  subscribeOutgoing: false,              // Receive outgoing messages in updates (default: false)
});
```

## Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed diagrams covering the startup flow, message lifecycle, middleware engine, WebSocket protocol, and security model.

## License

GPL-3.0

## Author

[ArnabXD](https://arnabxd.me)