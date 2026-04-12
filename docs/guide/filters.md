# Filters & Events

## Event Filters

Use `wa.on()` with a filter string to handle specific event types:

```ts
// Any message
wa.on("message", async (ctx) => { /* ... */ });

// Text messages only
wa.on("message:text", async (ctx) => {
  console.log(ctx.text); // string, guaranteed
});

// Image messages
wa.on("message:image", async (ctx) => {
  console.log(ctx.message.mimetype);
});
```

## Available Filters

| Filter | Event |
|--------|-------|
| `message` | Any message |
| `message:text` | Text messages |
| `message:image` | Image messages |
| `message:video` | Video messages |
| `message:audio` | Audio messages |
| `message:document` | Document messages |
| `message:sticker` | Sticker messages |
| `message:contact` | Contact cards |
| `message:location` | Location shares |
| `message_reaction` | Emoji reactions |
| `qr` | QR code for auth |
| `connected` | Connected to WhatsApp |
| `disconnected` | Disconnected |
| `receipt` | Delivery/read receipts |
| `presence` | Online/typing status |
| `group_join` | Member joined group |
| `group_leave` | Member left group |
| `group_update` | Group info changed |

## Multiple Filters

Pass an array to match multiple event types:

```ts
wa.on(["message:image", "message:video"], async (ctx) => {
  // handles both image and video messages
});
```

## Filter Constants

Use the `filters` object for autocomplete:

```ts
import { filters } from "@arnabxd/whatspurr";

wa.on(filters.text, async (ctx) => {
  await ctx.reply(ctx.text);
});
```

## Pattern Matching

Use `hears()` to match text messages against a regex:

```ts
wa.hears(/^hello/i, async (ctx) => {
  console.log(ctx.match); // RegExp match result
  await ctx.reply("Hi there!");
});
```

## Commands

Use `command()` to handle `/command` messages:

```ts
wa.command("start", async (ctx) => {
  await ctx.reply("Welcome!");
});

wa.command("echo", async (ctx) => {
  // /echo hello world → ctx.commandArgs = "hello world"
  await ctx.reply(ctx.commandArgs ?? "No arguments");
});
```

## Custom Filters

Use `filter()` with a predicate function:

```ts
// Only handle group messages
wa.filter(
  (ctx) => ctx.isGroup,
  async (ctx) => {
    await ctx.reply("This is a group!");
  }
);
```
