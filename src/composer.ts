import type { Context } from "./context.ts";
import type { FilterQuery, MiddlewareFn, NarrowContext, NextFn } from "./types.ts";

type MaybeArray<T> = T | T[];

export class Composer<C extends Context = Context> {
  private middlewares: MiddlewareFn<C>[] = [];

  /** Register one or more middleware functions */
  use(...fns: MiddlewareFn<C>[]): this {
    this.middlewares.push(...fns);
    return this;
  }

  /** Handle events matching a filter string (e.g. "message", "message:text", "group_join") */
  on<Q extends FilterQuery>(filter: MaybeArray<Q>, ...fns: MiddlewareFn<NarrowContext<C, Q>>[]): this {
    const filters = Array.isArray(filter) ? filter : [filter];
    return this.use(filterMiddleware(filters, fns as unknown as MiddlewareFn<C>[]));
  }

  /** Handle text messages matching a pattern */
  hears(trigger: RegExp | string, ...fns: MiddlewareFn<NarrowContext<C, "message:text">>[]): this {
    const regex = typeof trigger === "string" ? new RegExp(trigger) : trigger;
    return this.use((ctx, next) => {
      const text = ctx.text;
      if (text == null) return next();
      const match = text.match(regex);
      if (!match) return next();
      ctx.match = match;
      return run(fns as unknown as MiddlewareFn<C>[], ctx);
    });
  }

  /** Handle command messages (e.g. "/start", "/help args") */
  command(name: string, ...fns: MiddlewareFn<NarrowContext<C, "message:text">>[]): this {
    const prefix = `/${name}`;
    return this.use((ctx, next) => {
      const text = ctx.text;
      if (text == null) return next();
      if (text !== prefix && !text.startsWith(`${prefix} `)) return next();
      ctx.commandArgs = text.slice(prefix.length).trim();
      return run(fns as unknown as MiddlewareFn<C>[], ctx);
    });
  }

  /** Filter with a custom predicate */
  filter(predicate: (ctx: C) => boolean | Promise<boolean>, ...fns: MiddlewareFn<C>[]): this {
    return this.use(async (ctx, next) => {
      if (await predicate(ctx)) return run(fns, ctx);
      return next();
    });
  }

  /** Error boundary — catches errors from downstream middleware */
  errorBoundary(handler: (err: Error, ctx: C) => unknown | Promise<unknown>, ...fns: MiddlewareFn<C>[]): this {
    return this.use(async (ctx, next) => {
      try {
        if (fns.length > 0) {
          await run(fns, ctx);
        } else {
          await next();
        }
      } catch (err) {
        await handler(err instanceof Error ? err : new Error(String(err)), ctx);
      }
    });
  }

  /** Compile all registered middleware into a single function */
  middleware(): MiddlewareFn<C> {
    const mw = this.middlewares;
    return (ctx, next) => run(mw, ctx, next);
  }
}

/** Run a middleware chain (onion model) */
function run<C extends Context>(middlewares: MiddlewareFn<C>[], ctx: C, fallback?: NextFn): Promise<void> {
  let index = -1;

  function dispatch(i: number): Promise<void> {
    if (i <= index) return Promise.reject(new Error("next() called multiple times"));
    index = i;

    const fn = i < middlewares.length ? middlewares[i] : fallback;
    if (!fn) return Promise.resolve();

    return Promise.resolve(fn(ctx, () => dispatch(i + 1))).then(() => {});
  }

  return dispatch(0);
}

/** Create middleware that only runs if the event matches filter strings */
function filterMiddleware<C extends Context>(filters: FilterQuery[], fns: MiddlewareFn<C>[]): MiddlewareFn<C> {
  return (ctx, next) => {
    const eventType = ctx.eventType;
    const msgType = ctx.message?.type;

    for (const f of filters) {
      // Exact event match: "message", "qr", "connected", etc.
      if (f === eventType) return run(fns, ctx);

      // Subtype match: "message:text", "message:image", etc.
      if (f.includes(":")) {
        const [event, subtype] = f.split(":");
        if (event === eventType && subtype === msgType) return run(fns, ctx);
      }
    }

    return next();
  };
}
