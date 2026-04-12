---
layout: home

hero:
  name: whatspurr
  text: WhatsApp bots, the TypeScript way
  tagline: A grammY-style library powered by whatsmeow — familiar middleware, full type safety, zero browser hacks.
  image:
    src: /logo.svg
    alt: whatspurr
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: API Reference
      link: https://npmx.dev/package-docs/@arnabxd/whatspurr/
    - theme: alt
      text: GitHub
      link: https://github.com/ArnabXD/whatspurr

features:
  - icon: "\U0001F41E"
    title: grammY-style API
    details: Middleware, composers, context objects, filter queries — if you've built Telegram bots with <a href="https://grammy.dev" target="_blank">grammY</a>, you already know this.
  - icon: "\U0001F510"
    title: Powered by whatsmeow
    details: Battle-tested Go implementation of the WhatsApp Web protocol. No Puppeteer, no unofficial REST APIs.
  - icon: "\U0001F4E6"
    title: Zero Config
    details: The Go bridge binary auto-downloads on first run. Install the package and start writing handlers.
  - icon: "\U0001F9E9"
    title: Fully Typed
    details: Context narrowing gives your handlers exact types. <code>wa.on("message:text")</code> guarantees <code>ctx.text</code> is a string.
  - icon: "\U0001F680"
    title: Bun & Node
    details: Works on Bun and Node.js 21+. No runtime-specific APIs in the library — use whichever you prefer.
  - icon: "\U0001F527"
    title: Composable
    details: Split handlers into modules with Composer, scope error boundaries, nest middleware — scale from echo bot to production.
---

<style>
:root {
  --vp-home-hero-name-color: transparent;
  --vp-home-hero-name-background: -webkit-linear-gradient(120deg, #7c3aed 30%, #a855f7);
  --vp-home-hero-image-background-image: linear-gradient(-45deg, #7c3aed50 50%, #a855f750 50%);
  --vp-home-hero-image-filter: blur(44px);
}
</style>
