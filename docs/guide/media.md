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