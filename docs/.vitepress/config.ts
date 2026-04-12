import { defineConfig } from "vitepress";
import { withMermaid } from "vitepress-plugin-mermaid";

export default withMermaid(
  defineConfig({
    title: "whatspurr",
    description:
      "A grammY-style TypeScript library for WhatsApp, powered by whatsmeow",
    base: "/",
    cleanUrls: true,

    head: [
      ["link", { rel: "icon", type: "image/svg+xml", href: "/logo.svg" }],
    ],

    themeConfig: {
      nav: [
        { text: "Guide", link: "/guide/getting-started" },
        {
          text: "API Reference",
          link: "https://npmx.dev/package-docs/@arnabxd/whatspurr/",
        },
      ],

      sidebar: [
        {
          text: "Introduction",
          items: [
            { text: "Getting Started", link: "/guide/getting-started" },
            { text: "Configuration", link: "/guide/configuration" },
          ],
        },
        {
          text: "Core Concepts",
          items: [
            { text: "Middleware", link: "/guide/middleware" },
            { text: "Filters & Events", link: "/guide/filters" },
            { text: "Messaging", link: "/guide/messaging" },
            { text: "Media", link: "/guide/media" },
            { text: "Groups", link: "/guide/groups" },
          ],
        },
        {
          text: "Advanced",
          items: [
            { text: "Multi-Session", link: "/guide/multi-session" },
            { text: "Architecture", link: "/guide/architecture" },
          ],
        },
      ],

      socialLinks: [
        { icon: "github", link: "https://github.com/ArnabXD/whatspurr" },
        {
          icon: "npm",
          link: "https://www.npmjs.com/package/@arnabxd/whatspurr",
        },
      ],

      editLink: {
        pattern:
          "https://github.com/ArnabXD/whatspurr/edit/main/docs/:path",
      },

      footer: {
        message: "Released under the GPL-3.0 License.",
        copyright: "Copyright 2025 ArnabXD",
      },

      search: {
        provider: "local",
      },
    },
  }),
);
