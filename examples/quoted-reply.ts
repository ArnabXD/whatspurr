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

// Log all replies with info about the quoted message
wa.on("message", async (ctx) => {
  if (!ctx.isReply) return;

  const quoted = ctx.quotedMessage!;
  let description = `[${quoted.type ?? "unknown"}]`;

  if (quoted.type === "text") {
    description = `"${quoted.text}"`;
  } else if (quoted.type === "image" || quoted.type === "video") {
    description = `[${quoted.type}${quoted.caption ? `: ${quoted.caption}` : ""}]`;
  } else if (quoted.type === "contact") {
    description = `[contact: ${quoted.displayName}]`;
  } else if (quoted.type === "location") {
    description = `[location: ${quoted.latitude}, ${quoted.longitude}]`;
  }

  console.log(
    `${ctx.from} replied to ${quoted.sender}'s message (${quoted.messageId})`,
  );
  console.log(`  Quoted: ${description}`);
  console.log(`  Reply:  ${ctx.text ?? `[${ctx.message!.type}]`}\n`);

  await ctx.markRead();
});

// Reply to any "!quote" command with info about the quoted message
wa.command("quote", async (ctx) => {
  if (!ctx.isReply) {
    await ctx.reply("Reply to a message with /quote to see its info.");
    return;
  }

  const quoted = ctx.quotedMessage!;
  const lines = [
    `Message ID: ${quoted.messageId}`,
    `Sender: ${quoted.sender}`,
    `Type: ${quoted.type ?? "unknown"}`,
  ];

  if (quoted.type === "text") {
    lines.push(`Text: ${quoted.text}`);
  } else if (
    quoted.type === "image" ||
    quoted.type === "video" ||
    quoted.type === "document"
  ) {
    lines.push(`MIME: ${quoted.mimetype}`);
    if (quoted.caption) lines.push(`Caption: ${quoted.caption}`);
    if (quoted.type === "document" && quoted.filename)
      lines.push(`Filename: ${quoted.filename}`);
  }

  await ctx.reply(lines.join("\n"));
});

await wa.start();
console.log(
  "Listening for replies. Try replying to any message, or reply with /quote.",
);
console.log("Press Ctrl+C to stop.\n");

process.on("SIGINT", async () => {
  console.log("\nShutting down...");
  await wa.stop();
  process.exit(0);
});
