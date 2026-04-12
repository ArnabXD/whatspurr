# whatspurr

A [grammY](https://grammy.dev)-style TypeScript library for WhatsApp, powered by [whatsmeow](https://github.com/tulir/whatsmeow) (Go) via a WebSocket sidecar.

**[Documentation](https://whatspurr.arnabxd.me)** | **[API Reference](https://npmx.dev/package-docs/@arnabxd/whatspurr/)**

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
  console.log("\nScan this QR code in WhatsApp:\n");
  console.log(renderUnicodeCompact(ctx.qr.code));
});

// Connected
wa.on("connected", (ctx) => {
  console.log(`Connected as ${ctx.connected.jid}`);
});

// Disconnected
wa.on("disconnected", (ctx) => {
  console.log(`Disconnected: ${ctx.disconnected.reason}`);
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

## Multi-Session

Run multiple WhatsApp accounts from a single process using `WhatsAppManager`. All sessions share one Go bridge process and one SQLite database.

```ts
import { WhatsAppManager } from "@arnabxd/whatspurr";

const mgr = new WhatsAppManager({ sessionDir: "./session" });
await mgr.start();

// Connect a long-lived listener bot
const bot = await mgr.connect("support-bot");
bot.on("qr", (ctx) => console.log("Scan QR:", ctx.qr.code));
bot.on("message:text", async (ctx) => {
  await ctx.reply("Got it!");
});
await bot.start(); // registers handlers first, then connects

// Connect a sender, do work, disconnect (auth data is preserved)
const sender = await mgr.connect("bulk-sender");
await sender.start();
await sender.api.sendMessage("123@s.whatsapp.net", "Hello!");
await mgr.disconnect("bulk-sender"); // frees resources, can reconnect later

// List all sessions in the database
const sessions = await mgr.list();
// [{ name: "support-bot", jid: "...", connected: true },
//  { name: "bulk-sender", jid: "...", connected: false }]

// Reconnect later without QR (session data is in the DB)
const sender2 = await mgr.connect("bulk-sender");
await sender2.start();

// Remove a session entirely (logout + delete from DB)
await mgr.destroy("bulk-sender");

// Shutdown everything
await mgr.stop();
```

### Session lifecycle

| Method | What happens | Auth data | Can reconnect? |
|---|---|---|---|
| `connect(name)` | Prepares a WhatsApp instance with bridge listeners | - | - |
| `wa.start()` | Sends `connect_session`, starts whatsmeow goroutine | Preserved | - |
| `disconnect(name)` | Disconnects from WhatsApp, stops goroutine | Preserved | Yes (skip QR) |
| `destroy(name)` | Logout from WhatsApp, delete device from DB | Deleted | No (needs re-QR) |

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

See the [Architecture guide](https://whatspurr.arnabxd.me/guide/architecture) for detailed diagrams covering the startup flow, message lifecycle, middleware engine, WebSocket protocol, and security model.

## License

GPL-3.0

## Author

[ArnabXD](https://arnabxd.me)