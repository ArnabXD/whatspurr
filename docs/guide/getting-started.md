# Getting Started

## Installation

::: code-group
```bash [bun]
bun add @arnabxd/whatspurr
```
```bash [npm]
npm install @arnabxd/whatspurr
```
```bash [pnpm]
pnpm add @arnabxd/whatspurr
```
:::

The Go bridge binary is automatically downloaded on first `wa.start()`. To build from source instead:

```bash
bun run build:go
```

## Your First Bot

```ts
import { WhatsApp } from "@arnabxd/whatspurr";
import { renderUnicodeCompact } from "uqr";

const wa = new WhatsApp({
  sessionDir: "./session",
  logLevel: "info",
});

// QR code — render for scanning
wa.on("qr", (ctx) => {
  console.log("\nScan this QR code in WhatsApp:\n");
  console.log(renderUnicodeCompact(ctx.qr.code));
});

// Connected
wa.on("connected", (ctx) => {
  console.log(`Connected as ${ctx.connected.jid}`);
});

// Echo bot: reply to text messages
wa.on("message:text", async (ctx) => {
  console.log(`${ctx.from}: ${ctx.text}`);
  await ctx.reply(`Echo: ${ctx.text}`);
});

// Start
await wa.start();

// Graceful shutdown
process.on("SIGINT", async () => {
  await wa.stop();
  process.exit(0);
});
```

## How It Works

whatspurr runs a Go sidecar process that handles the WhatsApp Web protocol via [whatsmeow](https://github.com/tulir/whatsmeow). Your TypeScript code communicates with it over a local WebSocket connection.

```
┌──────────────┐   WebSocket (localhost)   ┌──────────────┐
│  Your TS Bot │ ◄──────────────────────►  │  Go Sidecar  │
│              │    JSON commands/events   │  (whatsmeow) │
└──────────────┘                           └──────────────┘
                                                  │
                                           WhatsApp Servers
```

- The Go binary is started and managed automatically
- Session data is persisted in SQLite — scan QR once, stay connected
- All communication is local, no external servers involved
