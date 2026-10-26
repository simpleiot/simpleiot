# Simple IoT UI

The Svelte admin UI, served by `siot` at `/`. It is a SvelteKit app built with
`adapter-static` into `build/`, which `ui/embed.go` embeds in the binary.

## Commands

From the repository root, after `. envsetup.sh`:

- `siot_build_ui` builds the UI (type checks included) and gzips the hashed
  assets. `siot_build_frontend`, and so every build and run function, calls it.
- `siot_test_ui` runs prettier and eslint, `svelte-check`, and the vitest unit
  tests.
- `siot_watch_ui` starts the Vite dev server at `http://localhost:5173/`. It
  proxies `/v1`, `/fonts`, the Elm UI at `/old`, and the NATS WebSocket to a
  running `siot`, on `SIOT_HTTP_PORT` or one above `SIOT_NATS_PORT`.
  `siot_watch` starts it along with the Elm and Go watchers.

In `ui/` itself: `npm run dev`, `npm run build`, `npm run check`, `npm test`,
`npm run lint`, and `npm run format`.

## Browser test

`e2e/` holds a Playwright smoke test: sign in, add a node, edit its description,
and see the edit saved after a reload. It is opt-in and not part of `siot_test`
or CI. `siot_test_ui_browser` builds siot, starts it with a fresh data directory
on scratch ports (`SIOT_E2E_NATS_PORT`, 15222 by default, and the HTTP port one
above), runs the test, and stops the server. Install a browser for it once with
`npx playwright install chromium`, or set `SIOT_E2E_CHANNEL=chrome` to use an
installed Chrome.

## Build output

`build/` is ignored by Git and committed only in the release commit, as the Elm
assets in `frontend/public/dist` are, so that `go install` produces a binary
with the UI in it. The release commit stages it with `git add -f -A ui/build`.

`npm install` writes a `go.mod` into `node_modules`, so the Go tools treat it as
a separate module and skip any Go sources an npm package ships.
