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

// Auto-post received text messages as status/story
wa.on("message:text", async (ctx) => {
  if (ctx.isFromMe) return;

  const text = ctx.text!;
  console.log(`${ctx.from}: ${text}`);

  try {
    await ctx.api.postTextStatus(text);
    console.log(`  → Posted text as status\n`);
  } catch (err) {
    console.error(`  → Failed to post status:`, err);
  }
});

// Auto-post received images as status/story
wa.on("message:image", async (ctx) => {
  if (ctx.isFromMe) return;

  const msg = ctx.message!;
  console.log(`${ctx.from}: [image${msg.caption ? `: ${msg.caption}` : ""}]`);

  try {
    const { path } = await ctx.downloadMedia();
    const data = await Bun.file(path).bytes();
    await ctx.api.postImageStatus(data, {
      caption: msg.caption,
      mimetype: msg.mimetype,
    });
    await Bun.file(path).delete();
    console.log(`  → Posted image as status\n`);
  } catch (err) {
    console.error(`  → Failed to post status:`, err);
  }
});

// Auto-post received videos as status/story
wa.on("message:video", async (ctx) => {
  if (ctx.isFromMe) return;

  const msg = ctx.message!;
  console.log(`${ctx.from}: [video${msg.caption ? `: ${msg.caption}` : ""}]`);

  try {
    const { path } = await ctx.downloadMedia();
    const data = await Bun.file(path).bytes();
    await ctx.api.postVideoStatus(data, {
      caption: msg.caption,
      mimetype: msg.mimetype,
    });
    await Bun.file(path).delete();
    console.log(`  → Posted video as status\n`);
  } catch (err) {
    console.error(`  → Failed to post status:`, err);
  }
});

await wa.start();
console.log("Listening — any received text, image, or video will be posted as your status.");
console.log("Press Ctrl+C to stop.\n");

process.on("SIGINT", async () => {
  console.log("\nShutting down...");
  await wa.stop();
  process.exit(0);
});
