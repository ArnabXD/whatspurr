# Middleware

whatspurr uses the same middleware pattern as [grammY](https://grammy.dev) and [Koa](https://koajs.com) — the onion model.

## Basics

Every handler is a middleware function that receives a `Context` and a `next` function:

```ts
wa.use(async (ctx, next) => {
  console.log("before");
  await next(); // run downstream middleware
  console.log("after");
});
```

Call `await next()` to pass control to the next middleware. If you don't call `next()`, downstream middleware won't run.

## Ordering

Middleware runs in the order it's registered:

```ts
// 1. Logging
wa.use(async (ctx, next) => {
  const start = Date.now();
  await next();
  console.log(`Handled in ${Date.now() - start}ms`);
});

// 2. Ignore messages from self
wa.use(async (ctx, next) => {
  if (ctx.isFromMe) return; // stop here, don't call next()
  await next();
});

// 3. Echo handler
wa.on("message:text", async (ctx) => {
  await ctx.reply(`Echo: ${ctx.text}`);
});
```

## Composers

Use `Composer` to organize handlers into modules:

```ts
import { Composer, type Context } from "@arnabxd/whatspurr";

const admin = new Composer<Context>();

admin.command("ban", async (ctx) => {
  // handle /ban
});

admin.command("kick", async (ctx) => {
  // handle /kick
});

// Register the composer
wa.use(admin.middleware());
```

## Error Boundaries

Catch errors from downstream middleware:

```ts
wa.errorBoundary(async (err, ctx) => {
  console.error("Handler error:", err.message);
  await ctx.reply("Something went wrong!");
});
```

Or scope it to specific handlers:

```ts
const safe = new Composer<Context>();

safe.errorBoundary(
  async (err, ctx) => {
    console.error(err);
    await ctx.reply("Error in admin module");
  },
  async (ctx, next) => {
    // risky middleware here
    await next();
  }
);

wa.use(safe.middleware());
```
