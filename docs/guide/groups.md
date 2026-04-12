# Groups

## Group Events

Listen for group info changes:

```ts
wa.on("group_join", async (ctx) => {
  // Fires when the bot itself is added to a group
  const { chat, participants } = ctx.groupJoin;
  console.log(`Bot joined group ${chat}`);
});

wa.on("group_update", async (ctx) => {
  // Fires when group name or topic changes
  const { chat, field, value } = ctx.groupUpdate;
  console.log(`Group ${chat} updated ${field}: ${value}`);
});
```

::: info Limitations
- `group_join` only fires when the **bot itself** joins a group, not when other participants join.
- `group_leave` is not yet implemented.
- `group_update` currently tracks `name` and `topic` changes only.
:::

## Group Info

Fetch group details:

```ts
const group = await wa.api.getGroupInfo("120363xxx@g.us");

console.log(group.name);
console.log(group.topic);

for (const p of group.participants) {
  console.log(`${p.jid} (admin: ${p.isAdmin})`);
}
```

## Filtering Group Messages

Handle messages only from groups:

```ts
wa.filter(
  (ctx) => ctx.isGroup,
  async (ctx, next) => {
    console.log(`Group message in ${ctx.chat}`);
    await next();
  }
);
```

Or only from DMs:

```ts
wa.filter(
  (ctx) => !ctx.isGroup,
  async (ctx) => {
    await ctx.reply("This is a DM!");
  }
);
```

::: info Coming Soon
Group management commands (create group, add/remove participants, invite links, etc.) are planned for a future release.
:::
