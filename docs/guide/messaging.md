# Messaging


## Available Methods

| Method | Quotes original? | Description |
|--------|:-:|-------------|
| `ctx.reply(text)` | Yes | Reply with text |
| `ctx.replyWithImage(data, options?)` | Yes | Reply with an image |
| `ctx.replyWithVideo(data, options?)` | Yes | Reply with a video |
| `ctx.replyWithAudio(data, options?)` | Yes | Reply with audio |
| `ctx.replyWithDocument(data, options?)` | Yes | Reply with a document |
| `ctx.send(text)` | No | Send text to the chat |
| `ctx.sendImage(data, options?)` | No | Send an image |
| `ctx.sendVideo(data, options?)` | No | Send a video |
| `ctx.sendAudio(data, options?)` | No | Send audio |
| `ctx.sendDocument(data, options?)` | No | Send a document |

## Quoting a Specific Message

By default, `ctx.reply*()` quotes the message that triggered the handler. You can quote any message by passing `quotedMessageId` explicitly:

```ts
wa.on("message:text", async (ctx) => {
  // Quote a different message by ID
  await ctx.send("Check out that message above!", {
    quotedMessageId: "ABCDEF123456",
    quotedSender: "1234567890@s.whatsapp.net",
  });
});
```

`quotedSender` is the JID of the person who sent the quoted message — required in groups so WhatsApp can display the correct name.

## Reply vs Send

whatspurr has two families of context methods for sending messages:

- **`ctx.reply*()`** — creates a **quoted reply** (the recipient sees your message linked to the original, like long-pressing a message and tapping "Reply" in WhatsApp)
- **`ctx.send*()`** — sends to the same chat **without quoting**

```ts
wa.on("message:text", async (ctx) => {
  // Quoted reply — appears as a reply to the user's message
  await ctx.reply("Got it!");

  // Plain message — no quote, just sent to the chat
  await ctx.send("Here's some extra info.");
});
```

## Sending via API

For sending to arbitrary JIDs (outside a handler context):

```ts
// Plain message
await wa.api.sendMessage("1234567890@s.whatsapp.net", "Hello!");

// Quoted reply to a specific message
await wa.api.sendMessage("1234567890@s.whatsapp.net", "Replying to this!", {
  quotedMessageId: "ABCDEF123456",
  quotedSender: "1234567890@s.whatsapp.net",
});
```