# Plan: Svelte Admin UI at /v2

**Branch:** `cbrake/master` **Branched from:** `8ded2b93`

## Context

The web UI is an Elm single-page application: 71 files, about 13,800 lines, on
Elm 0.19.1, `elm-spa`, and `elm-ui`. It has two routes (sign-in and the node
tree) and 45 node editors that are forms over points, sharing one input kit
(`UI/NodeInputs.elm`). Nodes and points already move through JavaScript:
`frontend/lib` (`simpleiot-js`) holds the binary codec and the NATS client, and
`frontend/public/main.js` owns the connection and talks to Elm over a port
protocol (`Api/Nats.elm`). Sign-in and node operations (add, delete, move,
mirror, duplicate, notify, generate key) go over HTTP.

`../ui-planning.md` (outside this repo) sets out the longer direction: a
SvelteKit UI, a framework-agnostic SDK, dashboards, custom UIs, and reuse of the
nautilus HMI kit. This plan takes the first step only: **the admin UI the Elm
app provides today, one for one, in Svelte, served at `/v2` beside the Elm UI.**
No dashboards, no SDK package, no nautilus dependency. Both UIs run against the
same server until the Svelte UI covers every node type, and Elm is removed in a
later plan.

Agreed improvements beyond one-for-one (everything else stays as the Elm UI
behaves):

- Deep links: a URL per node.
- Expanded nodes are remembered across reloads.
- A search box that filters the tree.
- Keyboard navigation in the tree.
- Dark mode.
- A guard against losing unsaved point edits.

Out of scope: dashboards, custom UI hosting, node type descriptors, nautilus
components, notification display, removing Elm.

## Goals

1. A Svelte 5 + SvelteKit app in `ui/`, built with `adapter-static`, embedded in
   the `siot` binary, and served at `/v2`.
2. Every page, node editor, and node operation the Elm UI has, with the same
   optimistic behavior and live updates over NATS.
3. The Elm compiler's guarantees replaced by checks in the build: `strict`
   TypeScript with no `any`, `svelte-check` as a build failure, point and node
   type constants checked against `data/schema.go`, and unit tests for the tree
   and point logic.
4. The agreed improvements above.
5. `siot_build`, `siot_test`, `siot_watch`, CI, and the release process all
   cover the new UI.

## Design Decisions

### Project layout and toolchain

- **`ui/` at the top level**, with its own `package.json`, `embed.go`, and build
  output in `ui/build/`. `frontend/` is untouched except for type declarations
  added to `frontend/lib`. When Elm is removed, `frontend/` goes away and `ui/`
  stays.
- **SvelteKit with `adapter-static`**, `ssr = false`, `prerender = false`, a
  `fallback: "index.html"` page, and `paths.base = "/v2"`. Routes: `/v2/` (the
  tree), `/v2/sign-in`, and `/v2/nodes/[id]` (a deep link into the tree).
  SvelteKit is kept over plain Vite + Svelte because the later plans (custom
  UIs, dashboards) are SvelteKit apps and should share the same shape.
- **TypeScript `strict`**, `noImplicitAny`, `svelte-check` in the build,
  `eslint` (with `eslint-plugin-svelte`) and `prettier` as in `frontend/lib`.
  `vitest` for unit tests.
- **Styling is plain CSS with custom properties**, no CSS framework. The light
  theme reproduces the current look (IBM Plex Sans from `/fonts/`, the colors in
  `UI/Style.elm`); the dark theme is the same tokens with dark values. The page
  follows `prefers-color-scheme` and a toggle in the header overrides it, stored
  in `localStorage`. Component styles are Svelte scoped `<style>` blocks, which
  Vite emits as CSS files, so the content security policy needs no change for
  styles.
- **Icons** come from `@lucide/svelte` (the maintained successor of Feather,
  which the Elm UI uses), tree-shaken, plus the custom SIOT icons in
  `UI/Icon.elm` ported as small SVG components.
- **Components take props and emit callbacks; only the page and the store talk
  to the server.** This is the rule from `ui-planning.md`, applied from the
  start so the editors are reusable in dashboards later and test without a
  server.

### Client layer

- **`simpleiot-js` stays where it is** and is a file dependency of `ui/`
  (`"simpleiot-js": "file:../frontend/lib"`). A hand-written
  `frontend/lib/siot-nats.d.ts` (with `"types"` in its `package.json`) gives it
  types. `nats.ws` is bundled by Vite, so `/v2` has no import map.
- **The connection manager is a TypeScript port of `public/main.js`**: connect
  with the sign-in token, retry every 5 s, `me()` on connect and reconnect,
  subscriptions reconciled to the set of visible nodes, points batched per
  animation frame with newest-wins merging, subscriptions dropped while the tab
  is hidden and everything refetched when it is shown again, and an
  authorization failure signing the user out. It exposes plain callbacks, not
  Svelte state, so it can be unit tested.
- **The tree logic is a port of `Utils/NodeTree.elm`** as pure functions over a
  `TreeNode` type (`anchor`, `node`, `expChildren`, `expDetail`, `loading`,
  `hasChildren`, `children`), with the tests in `frontend/tests/NodeTree.elm`
  ported to vitest. The store (`tree.svelte.ts`) holds the forest in `$state`
  and applies these functions. Fetch depths and the watch-subject rules stay as
  they are (`up.<anchor>.<id>.*.*` and `up.<anchor>.*.<id>.*.*` for every node
  on screen).
- **Point helpers** (`getText`, `getValue`, `getBool`, `updatePoints`,
  `clearText`, `filterDeleted`, `sort`, and the key/value and list helpers) are
  ported from `Api/Point.elm`, with its tests. Point and node type constants go
  in `ui/src/lib/siot/schema.ts`. A Go test in `data/` reads that file and
  checks every string constant it declares against the constants in
  `data/schema.go`, so a renamed point type fails `go test`.
- **HTTP operations** (`POST /v1/auth`, add, delete, move, mirror, duplicate,
  notify, generate key) are one small module over `fetch`, with the same request
  bodies as `Api/Node.elm` and `Api/Auth.elm`.
- **Sign-in state is shared with the Elm UI**: the same `localStorage` key
  (`storage`) and shape (`{ user: { token, email } }`), so one sign-in serves
  both UIs during the transition. Anything the Svelte UI remembers for itself
  (theme, expanded nodes) goes under its own key, `siot.ui`.

### Serving at /v2

- `api.App` routes `/v2` and `/v2/...` to a second file server over the `ui`
  embedded FS, with the prefix stripped. A path with no file falls back to
  `index.html`, so a deep link loads the app. Files under `_app/immutable/` are
  hashed, and get `Cache-Control: public, max-age=31536000, immutable` like the
  fonts. The existing gzip wrapper (`fsDecomp`) is reused.
- **Content security policy.** SvelteKit's fallback page has one inline script
  that starts the app. At startup the server reads `index.html` from the `ui`
  FS, hashes each inline `<script>` body, and puts the hashes in the
  `script-src` of responses under `/v2`, the way `importMapHash` is allowed
  today. A test like `TestImportMapHash` checks that the page has exactly one
  inline script and that the computed policy matches it. Everything else in the
  policy stays the same.
- In dev mode (`siot -dev`), `/v2` is served from `./ui/build` on disk, as `/`
  is served from `./frontend/public`.
- The embed directive is `//go:embed all:build`, because `go:embed` skips files
  and directories whose names start with `_` and SvelteKit's output directory is
  `_app`.

### Build, dev, test, and release

- `siot_build_ui` runs `npm run build` in `ui/` (which runs `svelte-check`
  first) and gzips the files under `ui/build/_app` with `gzip -n`, as
  `siot_build_frontend` does for `elm.js`. `index.html` stays uncompressed, so
  the server can read it for the policy hashes. `siot_build_frontend` calls
  `siot_build_ui` after the Elm build, so every existing build and run function
  picks it up.
- `siot_install_frontend_deps` adds `(cd ui && npm ci)`.
- `siot_test_frontend` adds `siot_test_ui`: `npm run lint`, `npm run check`, and
  `npm test` (vitest) in `ui/`.
- `siot_watch` adds `siot_watch_ui`, which runs `vite dev`. The Vite dev server
  proxies `/v1`, `/fonts`, and the WebSocket at `/` (the regex key `^/$`, which
  is free because the app lives under `/v2`) to the siot HTTP port, taken from
  `SIOT_HTTP_PORT` with the usual default. The app is then at
  `http://localhost:5173/v2/`. Production stays the embedded build.
- **`ui/build` is committed only at release**, like `frontend/public/dist`, so
  `go install` produces a binary with the UI in it. Hashed file names mean each
  release replaces the files; `git add -A ui/build` is used in that one commit.
  The Committing section of `CLAUDE.md` and the release notes in `envsetup.sh`
  and `.github/workflows/release.yml` say so.
- **Playwright is opt-in.** `ui/e2e/` holds a smoke test (sign in, expand the
  root, open a node, edit a description and save, see it change) and
  `siot_test_ui_browser` starts a siot with a temporary data directory on
  scratch ports, runs it, and stops the server. It is not part of `siot_test` or
  CI.

### Pages and components

- `+layout.svelte` renders the header (logo, connection badge, search box, theme
  toggle, sign out) and guards every route but sign-in: no user in storage
  redirects to `/v2/sign-in`, and a user on `/v2/sign-in` is sent to the tree,
  as `Shared.elm` and `Auth.elm` do.
- `NodeTree.svelte` renders the forest; `NodeRow.svelte` renders one node:
  indent, expand arrow or spinner, detail dot, mirror badge, the editor, the
  save/discard row when the node has edits, and the operation panel (add,
  message, delete, paste) when its detail is open. The raw view, the "new point"
  inputs, and the copy and paste flows carry over as they are, including the
  rule that a node with an owning parent type may only be mirrored elsewhere
  (`Node.owningParentType`).
- **Each node type registers itself; the tree page knows no types.** In the Elm
  app the per-type knowledge is spread through `Home_.elm` (the editor switch in
  `viewNode`, the `nodeDesc*` texts, the per-parent add lists in `viewAddNode`,
  `nodeTypesThatHaveChildNodes`), `Api/Node.elm` (`owningParentType`), and
  `Utils/NodeTree.elm` (the sort rules). In the Svelte app each type is one
  directory, `ui/src/lib/nodes/<type>/`, holding its editor component and an
  `index.ts` that exports a descriptor:

  ```ts
  export default defineNodeType({
  	type: "modbus",
  	label: "Modbus",
  	description: "Modbus RTU or TCP client or server",
  	icon: Bus,
  	parents: ["device", "group"], // where it can be added
  	hasChildren: true,
  	ownedBy: undefined, // "modbus" on modbusIo: elsewhere it may only be mirrored
  	sortKey: undefined, // the custom sort rules
  	editor: Editor,
  })
  ```

  `ui/src/lib/nodes/index.ts` builds the registry with
  `import.meta.glob("./*/index.ts", { eager: true })`, so adding a type is
  adding a directory and nothing else. The tree page and the store ask the
  registry (`editorFor(type)`, `addableUnder(parentType)`, `hasChildren(type)`,
  `ownerOf(type)`, `sortKey(type)`) and contain no type names. An unknown type
  gets `Raw.svelte`, and the "custom" entry in the add list stays. The add list
  for a parent is every type whose `parents` include the parent's type, in
  registry order, which replaces the hand-ordered lists in `viewAddNode`.

- **Editors are props-only components.** Each gets the props the Elm
  `NodeOptions` record carries (`node`, `parent`, `children`, `now`, `zone`,
  `expDetail`, `modified`, `copy`, `scratch`, `generatedToken`) and callbacks
  (`onEditPoints`, `onSendPoints`, `onEditScratch`, `onUploadFile`,
  `onGenerateKey`). The header line of every editor (icon, description,
  "(disabled)") is a shared component the editor renders.
- **The input kit** in `ui/src/lib/inputs/` mirrors `UI/NodeInputs.elm`: text,
  number (with the `Sanitize` float handling), checkbox, option (radio), on/off,
  counter with reset, button action and button action with text, list, key and
  value, paste button, and the schedule input (time, date, weekday, with the
  local/UTC conversion in `Utils/Time.elm` and `Utils/Iso8601.elm`). Every input
  writes through `onEditPoints` and never calls the server.
- **Errors** stay as one dismissable banner under the header, as today.

### The improvements

- **Deep links.** Opening a node's detail replaces the URL with
  `/v2/nodes/<id>`; closing it goes back to `/v2/`. Loading `/v2/nodes/<id>`
  fetches `nodes.all.<id>` (the server already scopes the parents to what the
  user may see), walks up the parents until it reaches an anchor, expands each
  node along the way, opens the detail, and scrolls the node into view. A node
  that is not in scope shows "not found" and the normal tree. A "copy link"
  button sits next to the existing "copy ID".
- **Remembered expansion.** The set of expanded nodes and open details, as
  `{ anchor, id }`, is kept in `localStorage` under `siot.ui` per user ID and
  restored on connect by the same fetches a reconnect makes. A node that no
  longer exists is dropped silently.
- **Search.** A box in the header filters the tree by description, type, and ID,
  case-insensitive. The first use in a session fetches each anchor to a large
  depth so the whole tree is searchable, after which filtering is local. Matches
  show with their ancestors; everything else is hidden. Clearing the box
  restores the expansion state from before the search. The watch list stays
  "nodes on screen", so a search does not subscribe to the whole tree.
- **Keyboard navigation.** The tree uses the ARIA `tree` and `treeitem` roles
  with a roving tabindex. Up and Down move between visible nodes, Right expands
  or moves into children, Left collapses or moves to the parent, Enter toggles
  the detail, and Escape cancels an open operation. Enter in a form saves and
  Escape discards, as today.
- **Dark mode.** As described under styling.
- **Unsaved-edits guard.** With edits pending, `beforeunload` asks before the
  tab closes or reloads, `beforeNavigate` asks before an in-app navigation, and
  sign out asks first.

## Changes to note in the changelog

- **A new web UI is available at `/v2`**, built with Svelte. It has the same
  functions as the UI at `/` plus deep links, remembered expansion, search,
  keyboard navigation, and dark mode. The UI at `/` is unchanged; both share one
  sign-in. The `/v2` UI replaces the UI at `/` in a later release.
- Nothing is incompatible. `siot_setup` now installs the `ui/` dependencies as
  well, so run it again after upgrading a checkout.

## Phases

### Phase 0: Scaffold, serving, and build

- [ ] Create `ui/` with SvelteKit, `adapter-static`, Svelte 5, TypeScript
      `strict`, `eslint`, `prettier`, `vitest`, and `svelte-check` in
      `npm run build`. Set `paths.base`, `ssr = false`, the fallback page. Add
      `ui/node_modules` and `ui/.svelte-kit` to `.gitignore` (the `node_modules`
      rule exists; check `.svelte-kit`).
- [ ] Theme tokens in `ui/src/app.css`: the light theme from `UI/Style.elm`, a
      dark theme, `prefers-color-scheme`, and the header toggle.
- [ ] Sign-in page (`POST /v1/auth`, same `localStorage` key and shape as Elm),
      the layout with the route guard, and a placeholder tree page that shows
      the signed-in user's email and a sign-out button.
- [ ] `ui/embed.go` (`all:build`), the `/v2` route in `api/server.go` with the
      SPA fallback, the immutable cache header, the policy hashes computed from
      `index.html`, and `-dev` serving from `./ui/build`. Tests: the hash test,
      a fallback test, and the existing `TestImportMapHash` still passing.
- [ ] `envsetup.sh`: `siot_build_ui`, `siot_test_ui`, `siot_watch_ui`, the Vite
      dev proxy, and the calls from `siot_install_frontend_deps`,
      `siot_build_frontend`, `siot_test_frontend`, and `siot_watch`.
      `.github/workflows/go.yml` needs no change if those functions cover it;
      confirm.
- [ ] `ui/README.md` with the commands, and the changelog entry.
- [ ] `siot_test` passes. Commit.

### Phase 1: Client layer

- [ ] `frontend/lib/siot-nats.d.ts` and `"types"` in its `package.json`; `ui/`
      depends on `simpleiot-js` as a file dependency.
- [ ] `ui/src/lib/siot/schema.ts` (point types, point values, node types) and
      the Go test in `data/` that checks it against `data/schema.go`.
- [ ] `ui/src/lib/siot/point.ts`: the `Point` type, the helpers, and the tests
      from `frontend/tests/Point.elm`.
- [ ] `ui/src/lib/siot/node.ts`: the `Node` type, `edgeRole`, and `description`.
      No type lists: those come from the registry.
- [ ] `ui/src/lib/nodes/registry.ts`: `defineNodeType`, the descriptor type, the
      glob-built registry, and the query functions, with a test over a stub set
      of types.
- [ ] `ui/src/lib/siot/tree.ts`: the port of `Utils/NodeTree.elm` with its
      tests.
- [ ] `ui/src/lib/siot/connection.ts`: the port of `main.js`, with tests for the
      batching and merge logic (the NATS client is mocked).
- [ ] `ui/src/lib/siot/http.ts`: the HTTP operations.
- [ ] `ui/src/lib/util/`: `duration.ts`, `iso8601.ts`, `time.ts` (schedules),
      `sanitize.ts`, with the tests from `frontend/tests/`.
- [ ] `siot_test` passes. Commit.

### Phase 2: The node tree

- [ ] `tree.svelte.ts` store: connect on load, fetch anchors, expand and
      collapse, apply points and edge points, fetch new children, refetch on
      reconnect and on tab visibility, keep the watch list in step, pending
      edits with optimistic save, and the error banner.
- [ ] `NodeTree.svelte` and `NodeRow.svelte` with `Raw.svelte` as the only
      editor, so every node type is already usable. Include the raw view toggle
      and "add point", the mirror badge, and the connection badge. The registry
      holds descriptors without editors for every type at this point, so the add
      lists, child expansion, owner rule, and sort order all work before any
      editor is ported.
- [ ] Node operations: add (the per-parent type list and the custom type),
      delete (with the primary, mirror, and plain wording), copy and paste
      (move, mirror, duplicate, and the owning-parent rule), send message, copy
      ID, and refetch of the parent after each.
- [ ] Deep links (`/v2/nodes/[id]`), the URL following the open detail, and
      "copy link".
- [ ] Unsaved-edits guard.
- [ ] Check against the Elm UI side by side on one server: add, edit, move,
      mirror, delete, and live points from a signal generator.
- [ ] Update `docs/ref/frontend.md` with a section on the Svelte UI and
      `docs/user/ui.md` with a note on `/v2`. Commit.

### Phase 3: Input kit and node editors

Port by hand, one file per Elm component, keeping each editor's fields, labels,
defaults, and conditional fields. Commit after each batch.

- [ ] 3a: the input kit (`ui/src/lib/inputs/`), the shared editor header, and
      the simple editors: Group, NetworkManager, NTP, SparkplugGroup, Shelly,
      Browser, SparkplugDevice, SparkplugNode, OneWire, Mqtt, CanBus, MqttSub,
      NetworkManagerConn, Db, NetworkManagerDevice, User, Provisioning,
      MessageService, Sync, Rule, ProvisioningFile.
- [ ] 3b: Iio, Modbus, IioChannel, Variable, EnrollToken (generate key, shown
      once), MqttDevice, OneWireIO, DeviceCred, File (upload, text and binary,
      and the provisioning status), Particle, SignalGenerator, Device, Gpio,
      Update.
- [ ] 3c: ModbusIO, Action and ActionInactive, Metrics, Gps, Condition (the
      schedule input), ShellyIO, SerialDev (sends points directly for its
      actions).
- [ ] Every type in the Elm editor map has a registered editor; a test checks
      the registry against the node types in `schema.ts`, and another checks
      that no file outside `ui/src/lib/nodes/` names a node type. Commit.

### Phase 4: Search, remembered expansion, keyboard

- [ ] Remembered expansion and open details per user.
- [ ] Search box, the one-time deep fetch, filtering with ancestors, and
      restoring the expansion state.
- [ ] Keyboard navigation with the ARIA tree roles.
- [ ] Docs for the three in `docs/user/ui.md`. Commit.

### Phase 5: Browser test, docs, and wrap-up

- [ ] `ui/e2e/` Playwright smoke test and `siot_test_ui_browser`, documented as
      opt-in in `ui/README.md` and `docs/ref/development.md`.
- [ ] `CLAUDE.md`: the `ui/` directory in Key Directories, the Frontend
      Architecture section, the Committing section for `ui/build`, and the build
      commands.
- [ ] Release process: `ui/build` committed in the release commit beside
      `frontend/public/dist`; update the comments in `envsetup.sh` and
      `release.yml`.
- [ ] Changelog entry final, plans index updated. Commit.

### Follow-ups (not in this plan)

- Remove the Elm UI and move the Svelte UI to `/` (with the font files moving to
  `ui/static/fonts`). `frontend/lib` moves to `ui/lib` in the same change,
  keeping the `simpleiot-js` package name; the fixture path in
  `data/point_fixture_test.go`, the `envsetup.sh` steps, and the docs that name
  the path move with it.
- Extract `@simpleiot/sdk` from `frontend/lib` and the `ui/src/lib/siot`
  modules, with Svelte and React bindings.
- Dashboards as nodes, custom UI hosting, nautilus faceplates, and node type
  descriptors, per `../ui-planning.md`.
- Notification display in the UI (deferred from the notifications plan).
- Drop `'unsafe-inline'` from `style-src` once the Elm UI is gone.

## Testing Strategy

- **Unit (vitest):** the tree, point, duration, time, and sanitize ports, each
  with the cases from `frontend/tests/`, plus the batching in the connection
  manager with a mocked client.
- **Static:** `svelte-check` and `eslint` fail the build; the Go test keeps
  `schema.ts` in step with `data/schema.go`.
- **Go:** the policy hash test, the `/v2` fallback and cache header tests.
- **Manual, each phase:** the Svelte UI and the Elm UI open side by side on one
  server (ports from the dev instance note in memory), checking the same action
  in both.
- **Browser (opt-in):** the Playwright smoke test.

## Open Questions

- SvelteKit's `appDir` could be renamed from `_app` to avoid the `all:` embed
  prefix; `all:build` is simpler and is the plan unless it causes trouble.
- Whether editors should be lazy-loaded per type. The registry makes it a local
  change (the descriptor's `editor` becomes `() => import(...)`), so the plan
  ships one bundle and revisits if the embedded size grows.
- Search depth for the one-time fetch: start at 20 levels and measure on a large
  tree.

## Key Files

- `frontend/public/main.js`, `frontend/src/Api/Nats.elm`: the connection and
  port protocol being ported.
- `frontend/src/Pages/Home_.elm`: the tree page, operations, and the add-node
  lists.
- `frontend/src/Utils/NodeTree.elm`, `frontend/tests/NodeTree.elm`: the tree
  logic and its tests.
- `frontend/src/UI/NodeInputs.elm`, `frontend/src/UI/Sanitize.elm`,
  `frontend/src/Utils/Time.elm`: the input kit.
- `frontend/src/Components/Node*.elm`: the 45 editors.
- `frontend/src/Api/Point.elm`, `frontend/src/Api/Node.elm`, `data/schema.go`:
  types and constants.
- `api/server.go`, `api/csp_test.go`, `server/server.go`, `server/fs-decomp.go`,
  `frontend/embed.go`: serving and embedding.
- `envsetup.sh`, `.github/workflows/go.yml`, `.github/workflows/release.yml`:
  build, test, and release.
- `docs/ref/frontend.md`, `docs/user/ui.md`: documentation to update.
