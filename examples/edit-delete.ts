import { WhatsApp } from "../src/index.ts";
import { renderUnicodeCompact } from "uqr";

const wa = new WhatsApp({
  sessionDir: "./session",
  logLevel: "info",
  sessionName: "account-2",
});

wa.on("qr", (ctx) => {
  console.log("\nScan this QR:\n");
  console.log(renderUnicodeCompact(ctx.qr!.code));
});

wa.on("connected", (ctx) => {
  console.log(`Connected as ${ctx.connected!.jid}\n`);
});

// !edit <new text> — reply to a bot message to edit it
wa.command("edit", async (ctx) => {
  if (!ctx.isReply) {
    await ctx.reply("Reply to one of my messages with !edit <new text>");
    return;
  }

  const newText = ctx.commandArgs?.trim();
  if (!newText) {
    await ctx.reply("Usage: !edit <new text>");
    return;
  }

  const quoted = ctx.quotedMessage!;
  await ctx.editMessage(quoted.messageId, newText);
  console.log(`Edited message ${quoted.messageId} to: ${newText}`);
});

// !delete — reply to a bot message to delete it for everyone
wa.command("delete", async (ctx) => {
  if (!ctx.isReply) {
    await ctx.reply("Reply to one of my messages with !delete");
    return;
  }

  const quoted = ctx.quotedMessage!;
  await ctx.deleteMessage(quoted.messageId);
  console.log(`Deleted message ${quoted.messageId}`);
});

// !test — sends a message you can then edit or delete
wa.command("test", async (ctx) => {
  const result = await ctx.reply("This is a test message. Reply to it with !edit or !delete.");
  console.log(`Sent test message: ${result.messageId}`);
});

await wa.start();
console.log("Commands:");
console.log("  !test    — send a test message");
console.log("  !edit    — reply to a bot message with new text");
console.log("  !delete  — reply to a bot message to delete it");
console.log("\nPress Ctrl+C to stop.\n");

process.on("SIGINT", async () => {
  console.log("\nShutting down...");
  await wa.stop();
  process.exit(0);
});
