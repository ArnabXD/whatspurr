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

### Media Options

```ts
interface MediaSendOptions {
  caption?: string;   // Caption text (image, video, document)
  filename?: string;  // Filename (document)
  mimetype?: string;  // MIME type (defaults: image/jpeg, video/mp4, audio/ogg)
}
```

::: tip
All media methods come in `reply` and `send` variants — see [Messaging](/guide/messaging) for the difference.
:::

## Downloading Media

When you receive a media message, use `ctx.downloadMedia()` to download and decrypt it to disk:

```ts
wa.on("message:image", async (ctx) => {
  // Download to the configured downloadDir (default: ./downloads)
  const { path, size } = await ctx.downloadMedia();
  console.log(`Saved ${size} bytes to ${path}`);

  // Or specify an explicit path
  const result = await ctx.downloadMedia("/tmp/photo.jpg");
});
```

### Download Directory

Configure where media files are saved by default:

```ts
const wa = new WhatsApp({
  downloadDir: "./media", // default: "./downloads"
});
```

When no explicit path is given, files are saved as `<downloadDir>/<uuid>.<ext>` with the extension inferred from the MIME type.

### Download via API

Download using a media reference directly:

```ts
// Default path (downloadDir/<uuid>.<ext>)
const result = await wa.api.downloadMedia(mediaRef);

// Explicit path
const result2 = await wa.api.downloadMedia(mediaRef, "/tmp/file.pdf");
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