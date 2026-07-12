# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# whatspurr

A grammY-style TypeScript library for WhatsApp, powered by whatsmeow (Go) via a WebSocket sidecar.

## Commands

```bash
bun run build          # tsc -p tsconfig.build.json → dist/
bun run typecheck
bun run lint / lint:fix
bun run format
bun run build:go        # go mod tidy + build → bin/bridge
bun run release:go      # cross-compile all platforms → dist/
bun run docs:dev / docs:build / docs:preview   # VitePress, docs/
bun run changelog / release   # changelogen
```

No test suite yet (`bun test` has nothing to run).

## Architecture

- `src/`: TS library — `WhatsApp` (extends `Composer`), middleware, `Context`, filters.
- `go/`: thin sidecar running whatsmeow, one JSON WebSocket on `127.0.0.1:<random port>`, single connection only (second connect gets HTTP 409).
- Bridge binary is auto-downloaded from GitHub releases on first `bridge.start()` (`src/bridge.ts` `downloadBinary`); resolution order: `config.binaryPath` → `bin/bridge<ext>` → download. Contributors build locally via `bun run build:go`.

## Bridge Protocol

1. TS spawns the Go binary (`--session-dir`, `--db-name`, `--download-dir`, `--log-level`, `--auto-presence`, `--subscribe-outgoing`); auth token passed via `BRIDGE_TOKEN` env var (not a flag, to avoid leaking in `ps`).
2. Go prints `ready 127.0.0.1:PORT` to stdout when its listener is up.
3. TS connects a WebSocket there, authenticating via `bridge-auth-<token>` subprotocol (constant-time compared on the Go side).
4. TS pings every 20s; unexpected close triggers reconnect with exponential backoff (1s → 30s cap).

Wire format (JSON):
```
Command:  {"id": "uuid", "session": "name", "method": "send_message", "params": {...}}
Response: {"id": "uuid", "result": {...}} or {"id": "uuid", "error": {"code": N, "message": "..."}}
Event:    {"type": "event", "session": "name", "event": "message", "data": {...}}
```

Commands time out client-side at 30s. Go runs them off a semaphore capped at 64 concurrent goroutines (`maxConcurrentCommands`, `go/session.go`). Read limit 135MB (100MB media, base64 overhead). Sessions persist via SQLite (`modernc.org/sqlite`, no CGo); session name→JID mapping lives in `sessionDir/sessions.json`, separate from whatsmeow's device store.

Session-scoped methods dispatch through `Session.handlers` (`go/commands.go` `buildCommandHandlers`). Lifecycle methods (`connect_session`, `disconnect_session`, `destroy_session`, `list_sessions`) are handled directly by `SessionManager`.

## Key Data Flows

**Send**: `ctx.reply()` / `wa.api.sendMessage()` → `Api` (`src/api.ts`) → `bridge.send(method, params, session)` → Go `commands.go` handler → whatsmeow.

**Receive**: whatsmeow event → `go/handler.go` → `SessionManager.sendEvent` → WS push → `Bridge` emits `"event"` → `WhatsApp.handleEvent` filters by session name → `Context` → Composer middleware chain (onion model, `src/composer.ts`).

**Multi-session** (`src/manager.ts`): `WhatsAppManager` owns one shared `Bridge`; `connect(name)` binds a `WhatsApp` to it (vs. standalone `new WhatsApp(config)`, which owns its own bridge process). `disconnect()` preserves auth data (reconnect without QR); `destroy()` logs out and deletes the device.

**Media** (`go/media.go`): image dimensions via stdlib `image.DecodeConfig` (+ `x/image/webp`); video dimensions by shelling to `ffprobe` if present. Both fall back to `(0, 0)` silently — not fatal.

## Project Structure

```
src/    whatsapp.ts manager.ts composer.ts context.ts bridge.ts api.ts types.ts filters.ts index.ts
go/     main.go (entry/flags/WS server) session.go (SessionManager/dispatch) handler.go (events→JSON) commands.go (commands→whatsmeow) media.go
scripts/  build-go.ts  release.ts
docs/     VitePress site
examples/ example bots
bin/      compiled Go binary (gitignored)
```

## Guidelines

- Bun over Node: `bun <file>`, `bun test`, `bun install`, `bun run`. Bun auto-loads `.env`.
- Native `WebSocket`, not `ws`. `child_process.spawn` for the Go binary.
- `src/` must run on both Bun and Node — no Bun-only APIs (`Bun.file`, `bun:sqlite`) there. `scripts/` can use Bun freely.
- Two tsconfigs: `tsconfig.json` (dev) vs `tsconfig.build.json` (emit, ESM extension rewrite).
- Biome, not ESLint. Line width 120, 2-space indent.
- Go: keep it a thin dispatch layer. `modernc.org/sqlite`, `github.com/coder/websocket`.

## Types Reference

JID formats: `"1234567890@s.whatsapp.net"` (DM), `"120363xxx@g.us"` (group)

Filters for `wa.on()`: `"message"`, `"message:text"`, `"message:image"`, `"message:video"`, `"message:audio"`, `"message:document"`, `"connected"`, `"disconnected"`, `"qr"`, `"message_reaction"`, `"receipt"`, `"presence"`, `"group_join"`, `"group_leave"`, `"group_update"` — or use `filters` constants.

`NarrowContext<C, FilterQuery>` powers type-narrowed context (`ctx.message`, `ctx.text`, etc.) after filtering.