import { WhatsAppManager } from "../src/index.ts";
import { renderUnicodeCompact } from "uqr";

const mgr = new WhatsAppManager({
  sessionDir: "./session",
  logLevel: "info",
});

await mgr.start();
console.log("Bridge started\n");

// --- Connect first account ---
console.log("=== Connecting account-1 ===");
const wa1 = await mgr.connect("account-1");

wa1.on("qr", (ctx) => {
  console.log("\n[account-1] Scan this QR:\n");
  console.log(renderUnicodeCompact(ctx.qr!.code));
});

await new Promise<void>((resolve) => {
  wa1.on("connected", (ctx) => {
    console.log(`[account-1] Connected as ${ctx.connected!.jid}\n`);
    resolve();
  });
  wa1.start(); // sends connect_session after listeners are set
});

// --- Connect second account ---
console.log("=== Connecting account-2 ===");
const wa2 = await mgr.connect("account-2");

wa2.on("qr", (ctx) => {
  console.log("\n[account-2] Scan this QR:\n");
  console.log(renderUnicodeCompact(ctx.qr!.code));
});

await new Promise<void>((resolve) => {
  wa2.on("connected", (ctx) => {
    console.log(`[account-2] Connected as ${ctx.connected!.jid}\n`);
    resolve();
  });
  wa2.start();
});

// --- Echo handler for both accounts ---
for (const [name, wa] of [
  ["account-1", wa1],
  ["account-2", wa2],
] as const) {
  wa.on("message:text", async (ctx) => {
    console.log(`[${name}] ${ctx.from}: ${ctx.text}`);
    await new Promise((r) => setTimeout(r, 2000));
    await ctx.markRead();
    await new Promise((r) => setTimeout(r, 1000));
    await ctx.sendTyping();
    await new Promise((r) => setTimeout(r, 2000));
    await ctx.reply(`Echo: ${ctx.text}`);
  });
}

console.log("Both accounts connected. Echoing messages with quoted replies...");
console.log("Press Ctrl+C to stop.\n");

process.on("SIGINT", async () => {
  console.log("\nShutting down...");
  await mgr.stop();
  process.exit(0);
});
