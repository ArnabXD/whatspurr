# Contacts & User Info

## Check if on WhatsApp

Validate phone numbers before sending messages:

```ts
const results = await wa.api.isOnWhatsApp(["+1234567890", "+0987654321"]);

for (const r of results) {
  console.log(`${r.query}: ${r.isIn ? "registered" : "not found"}`);
  if (r.isIn) {
    console.log(`  JID: ${r.jid}`);
  }
  if (r.verifiedName) {
    console.log(`  Business: ${r.verifiedName}`);
  }
}
```

Inside a handler, use the context helper:

```ts
wa.command("check", async (ctx) => {
  const phone = ctx.commandArgs?.trim();
  if (!phone) return;

  const results = await ctx.isOnWhatsApp([phone]);
  const r = results[0];
  if (r?.isIn) {
    await ctx.reply(`${phone} is on WhatsApp (${r.jid})`);
  } else {
    await ctx.reply(`${phone} is not on WhatsApp`);
  }
});
```

## Get User Info

Fetch status text, picture ID, and device list for one or more users:

```ts
const users = await wa.api.getUserInfo([
  "1234567890@s.whatsapp.net",
  "0987654321@s.whatsapp.net",
]);

for (const [jid, info] of Object.entries(users)) {
  console.log(`${jid}:`);
  console.log(`  Status: ${info.status}`);
  console.log(`  Picture ID: ${info.pictureId}`);
  console.log(`  Devices: ${info.devices.length}`);
  if (info.verifiedName) {
    console.log(`  Business: ${info.verifiedName}`);
  }
}
```

In a handler, `ctx.getUserInfo()` defaults to the message sender:

```ts
wa.command("info", async (ctx) => {
  const users = await ctx.getUserInfo(); // sender's info
  const [jid, info] = Object.entries(users)[0];
  await ctx.reply(`Status: ${info.status}\nPicture ID: ${info.pictureId}`);
});
```

## Profile Picture

Get the profile picture URL for any user or group:

```ts
const pic = await wa.api.getProfilePicture("1234567890@s.whatsapp.net");

if (pic) {
  console.log(`URL: ${pic.url}`);
  console.log(`ID: ${pic.id}`);
  console.log(`Type: ${pic.type}`); // "image" or "preview"
}
```

Request a thumbnail instead of full resolution:

```ts
const thumb = await wa.api.getProfilePicture(jid, { preview: true });
```

Skip the download if you already have the latest picture:

```ts
const pic = await wa.api.getProfilePicture(jid, { existingId: "known-id" });
// Returns null if the picture hasn't changed
```

In a handler, `ctx.getProfilePicture()` defaults to the current chat:

```ts
wa.command("pic", async (ctx) => {
  const pic = await ctx.getProfilePicture();
  if (pic) {
    await ctx.reply(`Profile picture: ${pic.url}`);
  } else {
    await ctx.reply("No profile picture set.");
  }
});
```

## Subscribe to Presence

Watch when a user comes online or goes offline:

```ts
// Subscribe first
await wa.api.subscribePresence("1234567890@s.whatsapp.net");

// Then listen for presence events
wa.on("presence", (ctx) => {
  const p = ctx.presence!;
  console.log(`${p.from}: ${p.type}`); // "available", "unavailable", etc.
  if (p.lastSeen) {
    console.log(`Last seen: ${new Date(p.lastSeen * 1000).toLocaleString()}`);
  }
});
```

In a handler, `ctx.subscribePresence()` defaults to the message sender:

```ts
wa.command("watch", async (ctx) => {
  await ctx.subscribePresence();
  await ctx.reply(`Watching presence for ${ctx.from}`);
});
```

::: info
You must call `subscribePresence` for each JID you want to monitor. Presence events will only fire for subscribed JIDs.
:::

## Business Profile

Get business details for WhatsApp Business accounts:

```ts
const biz = await wa.api.getBusinessProfile("1234567890@s.whatsapp.net");

if (biz) {
  console.log(`Address: ${biz.address}`);
  console.log(`Email: ${biz.email}`);
  console.log(`Timezone: ${biz.timezone}`);

  for (const cat of biz.categories) {
    console.log(`Category: ${cat.name}`);
  }

  for (const h of biz.businessHours) {
    console.log(`${h.dayOfWeek}: ${h.openTime}–${h.closeTime} (${h.mode})`);
  }
}
```

In a handler, `ctx.getBusinessProfile()` defaults to the message sender:

```ts
wa.command("biz", async (ctx) => {
  const profile = await ctx.getBusinessProfile();
  if (profile) {
    await ctx.reply(`Business: ${profile.address || "No address"}`);
  } else {
    await ctx.reply("Not a business account.");
  }
});
```

::: tip
`getBusinessProfile` returns `null` for non-business accounts — use this to check whether a JID is a business.
:::

## Context Helper Summary

| Method | Default target | Description |
|--------|---------------|-------------|
| `ctx.isOnWhatsApp(phones)` | — | Check phone numbers |
| `ctx.getUserInfo(jids?)` | Message sender | Get user info |
| `ctx.getProfilePicture(jid?)` | Current chat | Get profile picture URL |
| `ctx.subscribePresence(jid?)` | Message sender | Watch online/offline status |
| `ctx.getBusinessProfile(jid?)` | Message sender | Get business details |
