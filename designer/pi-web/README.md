# InduForge Pi Web

`@induforge/pi-web` is the embedded AI development workspace used by InduForge. It is a focused fork of
[`@agegr/pi-web`](https://github.com/agegr/pi-web) based on upstream tag `v0.8.8`.

## Product Boundary

The fork keeps Pi Agent sessions, message branches, model and provider configuration, user API keys, file
browsing, Git diff, tool calls, and project skills. It removes host-duplicated features:

- Pi branding header and local language/theme controls
- project selection and arbitrary directory browsing
- Plugins management and automatic user/project plugin discovery
- system prompt inspection and project/user `SYSTEM.md` overrides
- Git Worktree UI and API

The workspace is always `/workspace`. Skills are loaded only from `/workspace/.pi/skills`. Platform-owned,
read-only extensions may be loaded from `/opt/induforge/pi/extensions`; users cannot manage that directory
through Pi Web.

## Embedded Configuration

| Environment variable | Purpose | Default |
| --- | --- | --- |
| `PI_WEB_EMBEDDED` | Identifies the InduForge embedded deployment. | `1` in the workspace image |
| `PI_WEB_WORKSPACE_ROOT` | Fixed project workspace. | `/workspace` |
| `PI_WEB_ALLOWED_PARENT_ORIGINS` | Exact comma-separated Designer/dev_ide origins allowed to send host context. | Empty |
| `PI_WEB_PLATFORM_EXTENSIONS_DIR` | Optional read-only platform extension directory. | Unset |
| `PI_CODING_AGENT_DIR` | Shared Pi Agent configuration, credentials, and sessions. | Pi default |

Designer appends only `induforgeProjectId` to the iframe URL. After configuration loads, Pi Web emits
`INDUFORGE_PI_READY`. Designer responds with `INDUFORGE_PI_CONTEXT`, containing the project ID, fixed
workspace root, locale, and theme. Both sides validate the exact origin, message version, project ID, and
parent/iframe window reference.

## Development

Pi Web 源码作为 InduForge 根 pnpm workspace 的一部分维护，Node.js 24.19.0 或更高版本为必需环境。

```bash
pnpm install
pnpm typecheck:pi-web
pnpm test:pi-web
pnpm lint:pi-web
pnpm build:pi-web
pnpm --dir designer/pi-web pack --dry-run
```

Run locally with:

```bash
PI_WEB_WORKSPACE_ROOT=/workspace \
PI_WEB_ALLOWED_PARENT_ORIGINS=http://localhost:18603 \
pnpm --dir designer/pi-web dev:lan
```

The published package name is `@induforge/pi-web`; the `pi-web` executable name is retained for container
startup compatibility. Upstream copyright and the MIT license remain in [LICENSE](./LICENSE).
