# whatspurr

A grammY-style TypeScript library for WhatsApp, powered by whatsmeow (Go) via a WebSocket sidecar.

## Architecture

- **TypeScript library** (`src/`): grammY-style API with middleware, composers, context objects
- **Go sidecar** (`go/`): Thin bridge that runs whatsmeow and communicates via WebSocket over localhost TCP
- **Transport**: JSON messages over WebSocket on `127.0.0.1` (random port, single-client mode, auth token)
- **Distribution**: Contributors build Go locally. End users get prebuilt binaries via postinstall.

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

## Go Guidelines

- Go source lives in `go/`.
- Use `modernc.org/sqlite` (pure Go SQLite, no CGo) for session storage.
- Use `nhooyr.io/websocket` for the WS server.
- Keep Go code minimal — it's a thin dispatch layer mapping JSON commands to whatsmeow API calls.
- Build locally: `cd go && go build -o ../bin/bridge`
- Cross-compile: `GOOS=linux GOARCH=amd64 go build -o ../bin/bridge-linux-amd64`

## Project Structure

```
src/              TypeScript library (grammY-style API)
  whatsapp.ts     Main WhatsApp class (extends Composer)
  composer.ts     Middleware engine
  context.ts      Per-event context with reply helpers
  bridge.ts       Go process lifecycle + WS client
  api.ts          Direct API methods
  types.ts        Type definitions
  filters.ts      Event filter predicates
  index.ts        Public exports
go/               Go sidecar source
  main.go         Entry point, WS server on localhost
  session.go      WS connection management
  handler.go      whatsmeow events -> JSON
  commands.go     JSON commands -> whatsmeow calls
scripts/          Build & distribution scripts
  build-go.ts     Local dev build
  postinstall.ts  Download prebuilt binary for end users
  release.ts      Cross-compile for all platforms
bin/              Compiled Go binary (gitignored)
```