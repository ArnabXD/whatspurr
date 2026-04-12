# Media

## Sending Media

Reply with media using context helpers:

```ts
import { readFileSync } from "node:fs";

wa.command("photo", async (ctx) => {
  const image = readFileSync("./photo.jpg");
  await ctx.replyWithImage(image, { caption: "Check this out!" });
});

wa.command("doc", async (ctx) => {
  const pdf = readFileSync("./report.pdf");
  await ctx.replyWithDocument(pdf, {
    filename: "report.pdf",
    mimetype: "application/pdf",
  });
});
```

### Available Methods

| Method | Description |
|--------|-------------|
| `ctx.replyWithImage(data, options?)` | Send an image |
| `ctx.replyWithVideo(data, options?)` | Send a video |
| `ctx.replyWithAudio(data, options?)` | Send audio |
| `ctx.replyWithDocument(data, options?)` | Send a document |

### Media Options

```ts
interface MediaSendOptions {
  caption?: string;   // Caption text (image, video, document)
  filename?: string;  // Filename (document)
  mimetype?: string;  // MIME type (defaults: image/jpeg, video/mp4, audio/ogg)
}
```

## Sending via API

For sending to specific JIDs (not as a reply):

```ts
await wa.api.sendImage("1234567890@s.whatsapp.net", imageBuffer, {
  caption: "Hello!",
});

await wa.api.sendDocument("1234567890@s.whatsapp.net", pdfBuffer, {
  filename: "report.pdf",
  mimetype: "application/pdf",
});
```
