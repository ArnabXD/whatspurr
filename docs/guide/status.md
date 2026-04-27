# Status & Stories

## Setting Your "About" Text

Set the text status that appears on your profile (the "About" line):

```ts
await wa.api.setStatusMessage("Building cool things with whatspurr!");
```

## Posting Stories

Post stories (status updates) that your contacts can view in the Status tab.

### Text Story

```ts
await wa.api.postTextStatus("Hello from whatspurr!");
```

### Image Story

```ts
import { readFileSync } from "node:fs";

const image = readFileSync("./photo.jpg");
await wa.api.postImageStatus(image, { caption: "Check this out!" });
```

### Video Story

```ts
const video = readFileSync("./clip.mp4");
await wa.api.postVideoStatus(video, { caption: "Watch this!" });
```

::: tip
All media options from [Media](/guide/media) apply here too — `mimetype`, `viewOnce`, `width`, `height`, etc.
:::

## Status Privacy

Check who can see your status updates:

```ts
const privacy = await wa.api.getStatusPrivacy();

for (const entry of privacy) {
  console.log(`Type: ${entry.type}, Default: ${entry.isDefault}`);
  console.log(`JIDs: ${entry.list.join(", ")}`);
}
```

Privacy types:

| Type | Meaning |
|------|---------|
| `"contacts"` | All contacts can see your status |
| `"blacklist"` | All contacts **except** those in `list` |
| `"whitelist"` | **Only** contacts in `list` |

::: warning Experimental
Posting stories uses WhatsApp's `status@broadcast` internally. whatsmeow marks this as **experimental** — it may not work reliably for accounts with very large contact lists.
:::
