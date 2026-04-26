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
  caption?: string;    // Caption text (image, video, document)
  filename?: string;   // Filename (document)
  mimetype?: string;   // MIME type (defaults: image/jpeg, video/mp4, audio/ogg)
  viewOnce?: boolean;  // Send as view-once (image, video)
  width?: number;      // Width in pixels (image, video)
  height?: number;     // Height in pixels (image, video)
}
```

::: tip
All media methods come in `reply` and `send` variants — see [Messaging](/guide/messaging) for the difference.
:::

## View-Once Media

Send images or videos as view-once — the recipient can only view them once before they disappear:

```ts
wa.command("secret", async (ctx) => {
  const image = readFileSync("./secret.jpg");
  await ctx.replyWithImage(image, {
    viewOnce: true,
    caption: "This disappears after viewing!",
  });
});
```

### Detecting Received View-Once Messages

Incoming view-once media has `viewOnce: true` on the message object:

```ts
wa.on("message:image", async (ctx) => {
  if (ctx.message.viewOnce) {
    console.log("Received a view-once image");
  }
});
```

## Media Dimensions

Image and video dimensions (width/height) are **auto-detected** by the Go sidecar when sending — you don't need to pass them manually.

- **Images**: detected via Go's stdlib (JPEG, PNG, GIF, WebP)
- **Videos**: detected via `ffprobe` if installed, silently skipped otherwise

You can still override with explicit values:

```ts
await ctx.replyWithImage(data, { width: 1080, height: 1920 });
```

Received image/video messages include dimensions when the sender provided them:

```ts
wa.on("message:image", async (ctx) => {
  const { width, height } = ctx.message;
  if (width && height) {
    console.log(`Image is ${width}x${height}`);
  }
});
```

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

// View-once image
await wa.api.sendImage("1234567890@s.whatsapp.net", imageBuffer, {
  viewOnce: true,
});
```