# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# whatspurr

A grammY-style TypeScript library for WhatsApp, powered by whatsmeow (Go) via a WebSocket sidecar.

## Commands

```bash
# TypeScript
bun run build          # tsc -p tsconfig.build.json → dist/
bun run typecheck      # type-check without emit
bun run lint           # biome check .
bun run lint:fix       # biome check --fix .
bun run format         # biome format --write .
bun test               # run all tests (none yet)
bun test <file>        # run a single test file

# Go sidecar
bun run build:go       # build for local dev → bin/bridge
bun run release:go     # cross-compile for all platforms

# Docs (VitePress)
bun run docs:dev       # local docs dev server
bun run docs:build     # build static docs
bun run docs:preview   # preview built docs

# Release
bun run changelog      # generate changelog
bun run release        # changelog + version bump
```

## Architecture

- **TypeScript library** (`src/`): grammY-style API with middleware, composers, context objects
- **Go sidecar** (`go/`): Thin bridge that runs whatsmeow and communicates via WebSocket over localhost TCP
- **Transport**: JSON messages over WebSocket on `127.0.0.1` (random port, single-client mode, auth token)
- **Distribution**: Contributors build Go locally. End users get prebuilt binaries auto-downloaded on first `wa.start()` (handled in `bridge.ts`).

## Bridge Protocol

Startup sequence:
1. TS spawns the Go binary with `--session-dir`, `--db-name`, `--download-dir`, etc. Auth token is passed via `BRIDGE_TOKEN` env var (not a CLI flag — avoids leaking it in `ps`).
2. Go prints `ready 127.0.0.1:PORT` to stdout when ready.
3. TS reads stdout, extracts address, opens a WebSocket connection using the token as a subprotocol (`bridge-auth-<token>`) instead of a query parameter.

Wire format (all JSON):
```
Command:  {"id": "uuid", "session": "name", "method": "send_message", "params": {...}}
Response: {"id": "uuid", "result": {...}} or {"id": "uuid", "error": {"code": N, "message": "..."}}
Event:    {"type": "event", "session": "name", "event": "message", "data": {...}}
```

Commands time out after 30s. The bridge allows a maximum of 64 concurrent commands (goroutine pool). Sessions persist via SQLite; name→JID mapping is stored in `sessionDir/sessions.json`.

## Key Data Flows

**Sending a message**: `ctx.reply()` → `Api.sendMessage()` → `bridge.send("send_message", params, session)` → Go `commands.go` → `whatsmeow.Client.SendMessage()`

**Receiving an event**: whatsmeow fires event → `handler.go` → `manager.sendEvent(session, type, data)` → WebSocket push → `bridge.ts` listener demux → `WhatsApp` emits to middleware stack → `Context` created → user handlers run

**Multi-session**: `WhatsAppManager` creates a single `Bridge`, then creates `WhatsApp` instances that share it. All commands are tagged with a session name; all events carry a session name for demultiplexing.

## Project Structure

```
src/              TypeScript library (grammY-style API)
  whatsapp.ts     WhatsApp class (extends Composer, single or managed session)
  manager.ts      WhatsAppManager for multi-session orchestration
  composer.ts     Middleware engine
  context.ts      Per-event context with reply helpers
  bridge.ts       Go process lifecycle + WS client
  api.ts          Direct API methods (session-aware)
  types.ts        Type definitions
  filters.ts      Event filter predicates
  index.ts        Public exports
go/               Go sidecar source
  main.go         Entry point, WS server on localhost
  session.go      SessionManager + per-session client lifecycle
  handler.go      whatsmeow events -> JSON (session-tagged)
  commands.go     JSON commands -> whatsmeow calls (per-session)
scripts/          Build & distribution scripts
  build-go.ts     Local dev build
  release.ts      Cross-compile for all platforms
bin/              Compiled Go binary (gitignored)
```

## Runtime

Default to using Bun instead of Node.js.

- Use `bun <file>` instead of `node <file>` or `ts-node <file>`
- Use `bun test` instead of `jest` or `vitest`
- Use `bun install` instead of `npm install` or `yarn install` or `pnpm install`
- Use `bun run <script>` instead of `npm run <script>` or `yarn run <script>` or `pnpm run <script>`
- Bun automatically loads .env, so don't use dotenv.

## TS Guidelines

- Native `WebSocket` (works in both Bun and Node 21+). Don't use `ws`.
- `child_process.spawn` for launching the Go binary (cross-runtime compatible).
- Library code in `src/` must work on both Bun and Node.js. Don't use Bun-specific APIs (`Bun.file`, `bun:sqlite`, etc.) in library code.
- Scripts in `scripts/` can use Bun APIs freely.
- Use `bun test` with `import { test, expect } from "bun:test"` for tests.
- Two tsconfigs: `tsconfig.json` (dev, allows `.ts` imports) and `tsconfig.build.json` (emit, rewrites extensions for ESM compat).
- Linter: Biome (not ESLint). Line width 120, 2-space indent.

## Go Guidelines

- Go source lives in `go/`.
- Use `modernc.org/sqlite` (pure Go SQLite, no CGo) for session storage.
- Use `nhooyr.io/websocket` for the WS server.
- Keep Go code minimal — it's a thin dispatch layer mapping JSON commands to whatsmeow API calls.
- Build locally: `cd go && go build -o ../bin/bridge`
- Cross-compile: `GOOS=linux GOARCH=amd64 go build -o ../bin/bridge-linux-amd64`

## Types Reference

JID formats: `"1234567890@s.whatsapp.net"` (DM), `"120363xxx@g.us"` (group)

Filter strings for `wa.on()`: `"message"`, `"message:text"`, `"message:image"`, `"message:video"`, `"message:audio"`, `"message:document"`, `"connected"`, `"disconnected"`, `"qr"`, `"message_reaction"`, `"receipt"`, `"presence"`, `"group_join"`, `"group_leave"`, `"group_update"` — or use constants from `filters`.

`NarrowContext<C, FilterQuery>` provides type-narrowed context in middleware; this is what powers type-safe access to `ctx.message`, `ctx.text`, etc. after filtering.
