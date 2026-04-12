# Configuration

Pass options to the `WhatsApp` constructor:

```ts
const wa = new WhatsApp({
  sessionDir: "./session",
  dbName: "whatspurr.db",
  logLevel: "info",
  binaryPath: "/path/to/bridge",
  binaryRepo: "ArnabXD/whatspurr",
  binaryVersion: "v0.1.0",
  autoPresence: true,
  subscribeOutgoing: false,
});
```

## Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `sessionDir` | `string` | `"./session"` | Directory for session/auth data |
| `dbName` | `string` | `"whatspurr.db"` | SQLite database filename |
| `logLevel` | `string` | — | Go sidecar log level: `"debug"`, `"info"`, `"warn"`, `"error"` |
| `binaryPath` | `string` | auto-detect | Path to Go bridge binary. Skips auto-download |
| `binaryRepo` | `string` | `"ArnabXD/whatspurr"` | GitHub repo for binary downloads |
| `binaryVersion` | `string` | `"latest"` | Pin a specific release version |
| `autoPresence` | `boolean` | `true` | Send "available" presence on connect |
| `subscribeOutgoing` | `boolean` | `false` | Receive outgoing messages (sent by us) in event updates |

## Binary Management

By default, whatspurr automatically downloads the correct Go bridge binary for your platform on first `wa.start()`. Supported platforms:

- macOS (arm64, amd64)
- Linux (amd64, arm64)
- Windows (amd64)

To use a custom binary:

```ts
const wa = new WhatsApp({
  binaryPath: "./my-custom-bridge",
});
```

To build from source:

```bash
cd go && go build -o ../bin/bridge
```
