# Architecture

whatspurr is a TypeScript library backed by a Go sidecar. The TypeScript layer provides the developer-facing API (middleware, composers, context objects), while the Go sidecar runs the WhatsApp protocol via [whatsmeow](https://github.com/tulir/whatsmeow). They communicate over a WebSocket on localhost.

## High-Level Overview

```mermaid
graph TB
    subgraph User Code
        Bot[Your Bot Code]
    end

    subgraph TypeScript Library
        Mgr[WhatsAppManager]
        WA[WhatsApp]
        Composer[Composer]
        Ctx[Context]
        Api[Api]
        Bridge[Bridge]
    end

    subgraph Go Sidecar
        Main[main.go]
        SM[SessionManager]
        S1[Session 'bot1']
        S2[Session 'bot2']
        Cmds[commands.go]
        Handler[handler.go]
        DB[(SQLite)]
    end

    WS[WhatsApp Servers]

    Bot --> Mgr
    Bot --> WA
    Mgr --> WA
    WA --> Composer
    WA --> Api
    WA --> Ctx
    Ctx --> Api
    Api --> Bridge
    Bridge --> Main
    Bridge <--> SM
    SM --> S1
    SM --> S2
    S1 --> Cmds
    S2 --> Cmds
    Cmds --> Handler
    Main --> SM
    Main --> DB
    S1 <--> WS
    S2 <--> WS
```

### Components

| Component | Role |
|---|---|
| **WhatsAppManager** | Multi-session orchestrator. Owns one `Bridge`, creates `WhatsApp` instances on demand. |
| **WhatsApp** | Per-session entry point. Extends `Composer` with middleware, filters, and event handling. |
| **Composer** | grammY-style middleware engine (onion model). |
| **Context** | Per-event object with reply helpers, message accessors, and API shortcuts. |
| **Api** | Direct API methods (`sendMessage`, `sendImage`, etc.). Session-aware. |
| **Bridge** | Manages the Go process lifecycle and WebSocket connection. |
| **SessionManager** | Go-side manager. Routes commands/events by session name. |
| **Session** | Go-side per-session wrapper around a `whatsmeow.Client` goroutine. |

## Startup Flow

```mermaid
sequenceDiagram
    participant Bot as Bot Code
    participant Mgr as WhatsAppManager
    participant Bridge as Bridge
    participant Go as Go Sidecar
    participant SM as SessionManager
    participant WM as whatsmeow
    participant WS as WhatsApp Servers

    Bot->>Mgr: mgr.start()
    Mgr->>Bridge: bridge.start()
    Bridge->>Go: spawn(binary, args, env: BRIDGE_TOKEN)
    Go->>Go: Init shared SQLite store
    Go->>SM: NewSessionManager(container)
    Go->>Go: Start WS server on 127.0.0.1:random
    Go-->>Bridge: stdout: "ready 127.0.0.1:PORT"
    Bridge->>Go: WebSocket connect (subprotocol auth)
    Note over Go: No sessions started yet

    Bot->>Mgr: mgr.connect("bot1")
    Mgr->>Bridge: send("connect_session", {name: "bot1"})
    Bridge->>SM: route command
    SM->>WM: NewClient + Connect()

    alt First time
        WM-->>SM: QR events
        SM-->>Bridge: {session: "bot1", event: "qr"}
        Bridge-->>Bot: ctx.qr.code
        Note over WS: User scans QR
    else Returning session
        WM->>WS: Resume from SQLite
    end

    WM-->>SM: Connected
    SM-->>Bridge: {session: "bot1", event: "connected"}
```

## Message Flow

```mermaid
sequenceDiagram
    participant WS as WhatsApp Servers
    participant WM as whatsmeow
    participant Session as Session (Go)
    participant SM as SessionManager
    participant Bridge as Bridge (TS)
    participant WA as WhatsApp
    participant MW as Middleware
    participant Ctx as Context

    WS->>WM: Encrypted message
    WM->>Session: events.Message
    Session->>SM: sendEvent("bot1", "message", data)
    SM->>Bridge: {"type":"event","session":"bot1","event":"message","data":{...}}
    Bridge->>WA: Filtered by session name
    WA->>Ctx: new Context(eventData, api)
    WA->>MW: Run middleware chain

    MW->>Ctx: ctx.reply("Hello!")
    Ctx->>Bridge: send("send_message", params, "bot1")
    Bridge->>SM: {"session":"bot1","method":"send_message","params":{...}}
    SM->>Session: Route to "bot1"
    Session->>WM: client.SendMessage()
    WM->>WS: Encrypted outgoing
    WM-->>Bridge: {messageId: "..."}
```

## Middleware Engine

```mermaid
graph LR
    Event([Incoming Event]) --> MW1

    subgraph Onion Model
        MW1[use - logger] -->|next| MW2[on - message]
        MW2 -->|next| MW3[hears - /hi/]
        MW3 -->|next| MW4[command - help]
        MW4 -->|next| END[fallback]
    end

    MW3 -.->|match!| Reply([ctx.reply])
```

- Each middleware calls `next()` to pass to the next one
- Filters (`on`, `hears`, `command`) skip to `next()` if they don't match
- When a filter matches, it runs its handlers and stops the chain
- Each `WhatsApp` instance has its own independent middleware stack

## WebSocket Protocol

All commands and events are multiplexed over a single WebSocket. The `session` field routes to the correct whatsmeow client.

### Commands (TS to Go)

```json
{"id": "uuid", "session": "bot1", "method": "send_message", "params": {"to": "jid", "text": "hi"}}
```

**Session management:**
- `connect_session` — start a session goroutine
- `disconnect_session` — stop goroutine, keep auth
- `destroy_session` — logout + delete from DB
- `list_sessions` — list all devices in DB

**Per-session:**
- `send_message`, `send_image`, `send_video`, `send_audio`, `send_document`
- `send_reaction`, `download_media`, `get_group_info`
- `send_chat_presence`, `mark_read`, `set_presence`

### Events (Go to TS)

```json
{"type": "event", "session": "bot1", "event": "message", "data": {...}}
```

Events: `qr`, `connected`, `disconnected`, `message`, `message_reaction`, `receipt`, `presence`, `group_join`, `group_update`

### Responses

```json
{"id": "uuid", "result": {"messageId": "ABC123"}}
```

```json
{"id": "uuid", "error": {"code": 1003, "message": "missing 'text' parameter"}}
```

## Security

- **Localhost only** — WS server binds to `127.0.0.1`, never exposed to the network
- **Random port** — OS assigns an ephemeral port, communicated via stdout
- **Auth token** — 32-byte random token passed via `BRIDGE_TOKEN` env var (not a CLI flag); authenticated via `Sec-WebSocket-Protocol` subprotocol header, verified with constant-time compare
- **Single WS connection** — only one client can connect at a time
- **Bounded concurrency** — max 64 concurrent command handlers
- **Size limits** — 135 MB WS read limit, per-type media caps (16 MB images, 100 MB documents)
- **Input validation** — JID parsing, base64 pre-checks, DB name path traversal prevention, `download_media` path confined to configured download directory
- **Opaque errors** — generic errors to the client, detailed logs server-side only
