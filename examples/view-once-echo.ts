import { WhatsAppManager } from "../src/index.ts";
import { renderUnicodeCompact } from "uqr";

const mgr = new WhatsAppManager({
  sessionDir: "./session",
  logLevel: "info",
  subscribeOutgoing: false
});

await mgr.start();
console.log("Bridge started\n");

const wa = await mgr.connect("account-2");

wa.on("qr", (ctx) => {
  console.log("\nScan this QR:\n");
  console.log(renderUnicodeCompact(ctx.qr!.code));
});

await new Promise<void>((resolve) => {
  wa.on("connected", (ctx) => {
    console.log(`Connected as ${ctx.connected!.jid}\n`);
    resolve();
  });
  wa.start();
});

// Echo received images back as view-once
wa.on("message:image", async (ctx) => {
  const msg = ctx.message!;
  console.log(`${ctx.from}: [image${msg.viewOnce ? " (view-once)" : ""}]`);

  await ctx.markRead();
  const { path } = await ctx.downloadMedia();
  const data = await Bun.file(path).bytes();

  await ctx.replyWithImage(data, {
    viewOnce: true,
    caption: msg.caption,
    mimetype: msg.mimetype,
  });

  await Bun.file(path).delete();
  console.log(`Echoed image as view-once`);
});

// Echo received videos back as view-once
wa.on("message:video", async (ctx) => {
  const msg = ctx.message!;
  console.log(`${ctx.from}: [video${msg.viewOnce ? " (view-once)" : ""}]`);

  await ctx.markRead();
  const { path } = await ctx.downloadMedia();
  const data = await Bun.file(path).bytes();

  await ctx.replyWithVideo(data, {
    viewOnce: true,
    caption: msg.caption,
    mimetype: msg.mimetype,
  });

  await Bun.file(path).delete();
  console.log(`Echoed video as view-once`);
});

console.log("Listening — send an image or video to get it back as view-once.");
console.log("Press Ctrl+C to stop.\n");

process.on("SIGINT", async () => {
  console.log("\nShutting down...");
  await mgr.stop();
  process.exit(0);
});
