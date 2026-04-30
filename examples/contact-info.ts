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

// !check <phone> — check if a phone number is on WhatsApp
wa.command("check", async (ctx) => {
  const phone = ctx.commandArgs?.trim();
  if (!phone) {
    await ctx.reply("Usage: !check <phone number>\nExample: !check +1234567890");
    return;
  }

  const results = await ctx.isOnWhatsApp([phone]);
  const r = results[0];
  if (!r) {
    await ctx.reply(`No results for ${phone}`);
    return;
  }

  const lines = [
    `Phone: ${r.query}`,
    `On WhatsApp: ${r.isIn ? "Yes" : "No"}`,
    `JID: ${r.jid}`,
  ];
  if (r.verifiedName) lines.push(`Business: ${r.verifiedName}`);
  await ctx.reply(lines.join("\n"));
});

// !info — get user info for the message sender
wa.command("info", async (ctx) => {
  const users = await ctx.getUserInfo();
  const info = Object.entries(users)[0];
  if (!info) {
    await ctx.reply("Could not get user info.");
    return;
  }

  const [jid, data] = info;
  const lines = [
    `JID: ${jid}`,
    `Status: ${data.status || "(none)"}`,
    `Picture ID: ${data.pictureId || "(none)"}`,
    `Devices: ${data.devices.length}`,
  ];
  if (data.verifiedName) lines.push(`Business: ${data.verifiedName}`);
  await ctx.reply(lines.join("\n"));
});

// !pic [jid] — get profile picture for sender or a specific JID
wa.command("pic", async (ctx) => {
  const jid = ctx.commandArgs?.trim() || undefined;
  const pic = await ctx.getProfilePicture(jid);

  if (!pic) {
    await ctx.reply("No profile picture found.");
    return;
  }

  await ctx.reply(`Profile picture:\nURL: ${pic.url}\nID: ${pic.id}\nType: ${pic.type}`);
});

// !biz [jid] — get business profile for sender or a specific JID
wa.command("biz", async (ctx) => {
  const jid = ctx.commandArgs?.trim() || undefined;
  const profile = await ctx.getBusinessProfile(jid);

  if (!profile) {
    await ctx.reply("No business profile found (not a business account).");
    return;
  }

  const lines = [
    `Business: ${profile.jid}`,
    `Address: ${profile.address || "(none)"}`,
    `Email: ${profile.email || "(none)"}`,
  ];
  if (profile.categories.length > 0) {
    lines.push(`Categories: ${profile.categories.map((c) => c.name).join(", ")}`);
  }
  if (profile.timezone) {
    lines.push(`Timezone: ${profile.timezone}`);
  }
  if (profile.businessHours.length > 0) {
    lines.push("Hours:");
    for (const h of profile.businessHours) {
      lines.push(`  ${h.dayOfWeek}: ${h.openTime}–${h.closeTime} (${h.mode})`);
    }
  }
  await ctx.reply(lines.join("\n"));
});

// !watch — subscribe to presence updates for the message sender
wa.command("watch", async (ctx) => {
  await ctx.subscribePresence();
  await ctx.reply(`Subscribed to presence updates for ${ctx.from}`);
});

// Log presence events
wa.on("presence", (ctx) => {
  const p = ctx.presence!;
  const lastSeen = p.lastSeen ? ` (last seen: ${new Date(p.lastSeen * 1000).toLocaleString()})` : "";
  console.log(`[presence] ${p.from}: ${p.type}${lastSeen}`);
});

await wa.start();
console.log("Commands:");
console.log("  !check <phone>  — check if a number is on WhatsApp");
console.log("  !info           — get sender's user info");
console.log("  !pic [jid]      — get profile picture URL");
console.log("  !biz [jid]      — get business profile");
console.log("  !watch          — subscribe to sender's presence");
console.log("\nPress Ctrl+C to stop.\n");

process.on("SIGINT", async () => {
  console.log("\nShutting down...");
  await wa.stop();
  process.exit(0);
});