# Architecture

## High-Level Overview

```mermaid
graph TB
    subgraph User Code
        Bot[Your Bot Code<br><code>wa.command, wa.hears, wa.on</code>]
    end

    subgraph TypeScript Library
        WA[WhatsApp<br><i>extends Composer</i>]
        Composer[Composer<br><i>middleware engine</i>]
        Ctx[Context<br><i>per-event, reply helpers</i>]
        Api[Api<br><i>sendMessage, sendImage...</i>]
        Bridge[Bridge<br><i>process + WS lifecycle</i>]
    end

    subgraph Go Sidecar
        Main[main.go<br><i>WS server on 127.0.0.1</i>]
        Session[session.go<br><i>read pump, concurrency</i>]
        Cmds[commands.go<br><i>JSON cmd → whatsmeow</i>]
        Handler[handler.go<br><i>whatsmeow evt → JSON</i>]
        WM[whatsmeow<br><i>WhatsApp Web protocol</i>]
        DB[(SQLite<br>session store)]
    end

    WhatsAppServers[WhatsApp Servers]

    Bot --> WA
    WA --> Composer
    WA --> Api
    WA -->|creates per event| Ctx
    Ctx --> Api
    Api --> Bridge
    Bridge -->|spawn process| Main
    Bridge <-->|WebSocket JSON| Session
    Session --> Cmds
    Handler --> Session
    Cmds --> WM
    WM --> Handler
    Main --> Session
    Main --> DB
    WM <-->|E2E encrypted| WhatsAppServers
```

## Startup Flow

```mermaid
sequenceDiagram
    participant Bot as Bot Code
    participant WA as WhatsApp
    participant Bridge as Bridge
    participant Go as Go Sidecar
    participant WM as whatsmeow
    participant WS as WhatsApp Servers

    Bot->>WA: wa.start()
    WA->>Bridge: bridge.start()
    Bridge->>Go: spawn(binary, [--token, --session-dir, ...])
    Go->>Go: Init SQLite store
    Go->>Go: Start HTTP/WS server on 127.0.0.1:0
    Go-->>Bridge: stdout: "ready 127.0.0.1:PORT"
    Bridge->>Go: WebSocket connect (ws://127.0.0.1:PORT/?token=...)
    Go->>Go: Verify token, accept single client
    Go->>WM: client.Connect()

    alt First time (no session)
        WM->>Go: QR channel events
        Go-->>Bridge: event: "qr" {code: "..."}
        Bridge-->>WA: event: "qr"
        WA-->>Bot: ctx.eventType === "qr"
        Note over Bot: Display QR code
        Note over WS: User scans QR
        WM-->>Go: Connected event
    else Returning session
        WM->>WS: Resume session from SQLite
        WM-->>Go: Connected event
    end

    Go-->>Bridge: event: "connected" {jid: "..."}
    Bridge-->>WA: event: "connected"
    WA-->>Bot: ctx.eventType === "connected"
```

## Message Flow (Incoming)

```mermaid
sequenceDiagram
    participant WS as WhatsApp Servers
    participant WM as whatsmeow
    participant Handler as handler.go
    participant Session as session.go
    participant Bridge as Bridge (TS)
    participant WA as WhatsApp
    participant MW as Middleware Chain
    participant Ctx as Context

    WS->>WM: Encrypted message
    WM->>Handler: *events.Message
    Handler->>Handler: Skip if IsFromMe<br>and !subscribeOutgoing
    Handler->>Handler: Parse message type<br>(text/image/video/...)
    Handler->>Session: sendEvent("message", data)<br>includes isFromMe flag
    Session->>Bridge: WS JSON: {"type":"event","event":"message","data":{...}}
    Bridge->>WA: emit("event", eventData)
    WA->>Ctx: new Context(eventData, api)
    WA->>MW: run middleware chain

    Note over MW: on("message") → filter check<br>hears(/regex/) → match check<br>command("/cmd") → prefix check

    MW->>Ctx: ctx.reply("Hello!")
    Ctx->>Bridge: api.sendMessage(chat, text)
    Bridge->>Session: WS JSON: {"id":"uuid","method":"send_message","params":{...}}
    Session->>Handler: handleCommand(cmd)
    Handler->>WM: client.SendMessage(jid, msg)
    WM->>WS: Encrypted outgoing message
    WM-->>Handler: response {ID: "..."}
    Handler-->>Session: Response{Result: {messageId}}
    Session-->>Bridge: WS JSON: {"id":"uuid","result":{"messageId":"..."}}
    Bridge-->>Ctx: {messageId: "..."}
```

## Middleware Engine

```mermaid
graph LR
    Event([Incoming Event]) --> MW1

    subgraph Onion Model
        MW1[use&#40;logger&#41;] -->|next&#40;&#41;| MW2[on&#40;'message'&#41;]
        MW2 -->|next&#40;&#41;| MW3[hears&#40;/hi/&#41;]
        MW3 -->|next&#40;&#41;| MW4[command&#40;'help'&#41;]
        MW4 -->|next&#40;&#41;| END[fallback]
    end

    MW3 -.->|match!| Reply([ctx.reply])
```

The middleware follows an **onion model** (like Koa/grammY):
- Each middleware calls `next()` to pass control to the next one
- Filters (`on`, `hears`, `command`) skip to `next()` if they don't match
- When a filter matches, it runs its handlers and stops the chain

## WebSocket Protocol

```mermaid
graph LR
    subgraph "TS → Go (Commands)"
        C1["send_message {to, text}"]
        C2["send_image {to, data, mimetype}"]
        C3["send_reaction {to, messageId, emoji}"]
        C4["get_group_info {jid}"]
        C5["set_presence {type}"]
        C6["send_chat_presence {to, state, media}"]
        C7["mark_read {chat, sender, messageIds}"]
    end

    subgraph "Go → TS (Events)"
        E1["qr {code}"]
        E2["connected {jid}"]
        E3["message {id, from, chat, type, ...}"]
        E4["receipt {from, messageIds, type}"]
        E5["presence {from, type}"]
        E6["group_join / group_update"]
    end
```

### Command format
```json
{"id": "uuid", "method": "send_message", "params": {"to": "jid@s.whatsapp.net", "text": "hello"}}
```

### Response format
```json
{"id": "uuid", "result": {"messageId": "ABC123"}}
```

### Event format
```json
{"type": "event", "event": "message", "data": {"id": "...", "from": "...", "text": "...", "isFromMe": false}}
```

## Security Model

```mermaid
graph TB
    subgraph Transport Security
        A[127.0.0.1 only] --> B[Random port]
        B --> C[64-byte auth token]
        C --> D[Single-client mode]
    end

    subgraph Resource Limits
        E[64 concurrent commands max]
        F[5 min WS read timeout]
        G[140 MB WS read limit]
        H[Per-type media size limits<br>16 MB image/video/audio<br>100 MB document]
    end

    subgraph Input Validation
        I[JID parsing via whatsmeow]
        J[Base64 size pre-check]
        K[DB name path traversal check]
        L[Listen addr regex validation]
    end

    subgraph Error Handling
        M[Generic errors to client]
        N[Detailed logs server-side only]
        O[30s command timeout in TS]
        P[Level-aware logging via waLog]
    end
```
