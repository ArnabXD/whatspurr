# Groups

## Group Events

Listen for group membership and info changes:

```ts
wa.on("group_join", async (ctx) => {
  const { chat, participants } = ctx.groupJoin;
  for (const jid of participants) {
    await wa.api.sendMessage(chat, `Welcome, ${jid}!`);
  }
});

wa.on("group_leave", async (ctx) => {
  const { chat, participants } = ctx.groupLeave;
  console.log(`${participants.join(", ")} left ${chat}`);
});

wa.on("group_update", async (ctx) => {
  const { chat, field, value } = ctx.groupUpdate;
  console.log(`Group ${chat} updated ${field}: ${value}`);
});
```

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
