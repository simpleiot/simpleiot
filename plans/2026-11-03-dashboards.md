# Plan: Dashboards as nodes

**Branch:** `cbrake/master` **Branched from:** `a28a2902`

## Context

`../dashboards.md` (outside this repo) is the design: a dashboard is a tree of
`dashboard`, `widget`, and `link` nodes with ordinary points; widgets bind to
points and never compute; a reflowing grid and a scaled canvas are the two
layout hosts; links are graphical, anchored to ports, and imply no behavior; and
one node reference picker, backed by a server search, serves dashboard bindings
and rule editing alike. This plan turns that design into phases against the
Svelte UI in `ui/`, which the
[Svelte admin UI plan](2026-10-23-svelte-admin-ui.md) finished.

What exists today that the plan builds on:

- The node request `nodes.<parent>.<id>` (`store/store.go`, `nodesRequest`)
  takes `nodeType`, `depth`, and `tombstone` as points in the payload and walks
  the edge cache (`store/jetstream.go`, `getNodesDepth`, capped at
  `maxNodesDepth`). The user namespace `u.<anchor>.<user>.nodes.<parent>.<id>`
  (`handleUserRequest`) refuses targets outside the anchor and trims replies to
  scope.
- The Svelte UI registers node types as directories under `ui/src/lib/nodes/`
  with a descriptor (`registry.ts`) and a props-only editor (`types.ts`,
  `EditorProps`). Editors get `findNode`, which looks only at nodes the page has
  loaded, and the tree's search does a one-time deep fetch of 20 levels.
- The rule editors (`condition/Editor.svelte`, `action/Editor.svelte`) take the
  target node ID as text and show its description when the node happens to be
  loaded, with a copy and paste button.
- The header (`ui/src/lib/components/Header.svelte`) has a logo, a search slot,
  theme, and sign out; routes are `/`, `/nodes/[id]`, and `/sign-in`.
- The nautilus HMI kit is checked out at `../other-projects/nautilus/hmi` (0.6.0
  locally; `@joyautomation/nautilus-hmi` 0.8.0 on npm). Its `src/lib/mimic.ts`
  holds the port, anchoring, and routing helpers, and its faceplates are Svelte
  5, props-only, and token-themed.

Out of scope: historical data and trends over stored points, custom UI hosting
and third-party components, kiosk playlists, semantic zoom, the Elm UI.

## Goals

1. Three node types, `dashboard`, `widget`, and `link`, classified in
   `data/edge_role.go`, mirrored in `schema.ts`, with editors in the tree.
2. A `search` parameter on the node request, scoped by the anchor, returning
   matches with their ancestors.
3. A node reference picker in the input kit that searches the server, stores the
   ID, shows the description with a link, and offers the chosen node's point
   types and keys. The rule editors use it.
4. A dashboard viewer at `/dash/[id]` with a grid host, a dashboard list, and
   Dashboards and Nodes entries in the header; a home dashboard per user.
5. Edit mode: drag and resize on the grid, a widget panel that is the widget
   node's editor, add widget with `auto` component selection.
6. A canvas host with equipment widgets, ports, and links, from the nautilus
   mimic helpers and faceplates.
7. Relative bindings and nested dashboards.
8. Docs, changelog, and `siot_test` green at every phase.

## Design Decisions

### Points

New constants in `data/schema.go`, mirrored in `ui/src/lib/siot/schema.ts`
(`data/schema_ts_test.go` checks the pair):

```
Node types   dashboard, widget, link

dashboard    layout (grid|canvas), cols, canvasWidth, canvasHeight
widget       component, x, y, w, h
             prop     key=<name>     static display option
             bindNode key=<prop>     node ID
             bindType key=<prop>     point type
             bindKey  key=<prop>     point key
             bindPath key=<prop>     relative path (phase 5)
             port     key=<name>     "x,y,dir" override
link         from, to               "<widgetID>.<port>"  (from exists already)
             routing (direct|orthogonal)
             vertex   key=<index>    "x,y" interior vertices
             bind*    as widget, for flowing and rate
user         homeDashboard          node ID
request      search, limit          on nodes.<anchor>.all
```

Grid units are columns and rows; canvas units are canvas pixels. `x`, `y`, `w`,
`h` are plain numbers in both, and the host decides what they mean.

### Edge roles

`dashboard`, `widget`, and `link` go in `treeScopedTypes`: no client runs for
them, and several placements of a dashboard are meaningful. `widget` and `link`
are owned by `dashboard` in `nodeTypeOwners`.

### Server search

One more case in `nodesRequest`: when a `search` point is present and the
subject is `nodes.<anchor>.all`, the store calls
`getNodesSearch(anchor, query, limit)` instead of `getNodesDepth`. It walks the
edge cache breadth first from the anchor, bounded by `maxNodesDepth` and a visit
cap, reads each node's description through the same lookup `getNodes` uses, and
keeps nodes whose description contains the query, case folded, with prefix
matches ranked ahead of substring matches, then an exact ID match, then a type
match. `limit` defaults to 20 and is capped at 100. For each match the reply
also carries its ancestors up to the anchor, following `edgeCache.Parents`, so
the UI can show a path; `parent` on each `NodeEdge` places it.

Scope needs no new code: `handleUserRequest` already refuses a parent outside
the anchor, and the walk starts at the anchor. The reply trimming that
`handleUserRequest` applies to depth requests applies here too.

### The picker

`NodeRefInput.svelte` in `ui/src/lib/inputs/`, props-only like the rest of the
kit:

- `value` (the stored ID), `onChange(id)`, `search(query) => Promise<Match[]>`,
  `resolve(id) => Promise<Node | undefined>`, `label`.
- Empty: a text field. Typing debounces 150 ms and lists matches, each with
  description, type label from the registry, and the ancestor path. Arrow keys
  and Enter choose; Escape clears.
- Set: a chip with the description, the first eight characters of the ID, a link
  to `/nodes/<id>`, and a clear button. An ID that does not resolve shows the ID
  with a "not found" mark.

`EditorProps` gains `searchNodes` and `resolveNode`, supplied by the tree page
from the store. `TreeStore` gets `searchNodes(query, limit)` fanning out to each
anchor and merging, and `resolveNode(id)` through `nodes.all.<id>` with a cache
keyed by ID that the live point subscription refreshes when a description
changes. `frontend/lib/siot-nats.js` gets
`searchNodes(anchor, query, { limit })` and its `.d.ts` entry.

The condition and action editors replace their `nodeID` text field and paste
button with the picker. Once a node is chosen, the point type and key inputs
become `OptionInput`s listing the types and keys that node's points carry, with
the current list's entries kept, and a free-text fallback.

### Widget registry and `auto`

Widgets register the way node types do: a directory `ui/src/lib/widgets/<name>/`
with an `index.ts` descriptor (`name`, `label`,
`host: "grid" | "canvas" | "both"`, `ports`, `props` with defaults, `bindings`
with the prop names the component accepts) and a `Widget.svelte` that takes
resolved values as props. `ui/src/lib/widgets/index.ts` globs them, as
`nodes/index.ts` does.

`auto` maps a node type to a widget. The repo's rule is that nothing outside
`src/lib/nodes` steers behavior by a type name, so the mapping is a `widget`
field on `NodeTypeDescriptor`, set per type directory, and `auto` reads it from
the registry. A type without one falls back to the value widget.

### Bindings at runtime

`ui/src/lib/dash/bindings.svelte.ts` turns a widget node's `bind*` points into a
map of prop to live value. It subscribes through the store's existing
`up.<anchor>` subscription set, so no new connection code, and reports three
states per binding: live, stale (no point yet), and dead (node unresolved).
Widgets receive values and states as props and render the dead state themselves.

### Hosts

- **Grid.** A CSS grid of `cols` columns and fixed row height. Below 900px it
  becomes one column in tree order. Drag and resize are a small pointer-event
  implementation of our own writing `x`, `y`, `w`, `h`; gridstack stays an open
  question until the simple version proves short.
- **Canvas.** A plane of `canvasWidth` by `canvasHeight` scaled to the
  container, with pan and pinch-zoom, after `Mimic.svelte`. Below 900px it
  becomes a wrapping grid of cards, one per widget, each opening its faceplate.
  Links render in an SVG under the widgets with the endpoints derived from ports
  through the helpers in `mimic.ts`.

### Nautilus

Vendor, with attribution, into `ui/src/lib/vendor/nautilus/`: `mimic.ts` (ports,
anchoring, routing), the `Tank`, `Gauge`, `Pump`, `Valve`, `Pipe`, `Sparkline`,
and `StatusPill` faceplates, and `TrendBuffer` for live trends. The theme tokens
map onto `ui/src/app.css` rather than importing `theme.css`. Vendoring over a
dependency because the kit is at 0.x, pulls an SSE client the UI does not use,
and the pieces taken are small. Revisit when it reaches 1.0.

### Navigation

- Header gains Dashboards and Nodes links; on narrow screens they are tabs at
  the bottom.
- `/dash` lists dashboards found with `nodeType=dashboard` under each anchor.
- `/dash/[id]` is the viewer, with an Edit toggle in the header and `?kiosk`
  hiding the chrome.
- The `dashboard` editor in the tree shows its description, layout fields, and
  an Open button.
- Tapping a widget in view mode opens the bound node's faceplate: `NodeEditor`
  with the registry editor, as a `<dialog>` above 900px and as `/nodes/[id]`
  below it.
- `/` redirects to the user's `homeDashboard` when set, else `/dash` when any
  dashboard is visible, else the tree.

### Relative bindings

`bindPath.<prop>` holds `<type>/<description>` segments from the dashboard's
parent on the route the user opened it through (`/dash/[id]?via=<parentID>`,
defaulting to the first parent). The resolver walks children by type and
description, fetching through the store. The picker offers "relative to the
dashboard's parent" when the chosen node is below that parent and writes
`bindPath` instead of `bindNode`.

## Changes to note in the changelog

- Three new node types and a `search` parameter on the node request; nothing
  changes for existing deployments.
- `/` now lands on a dashboard for users with one set; the tree remains at
  `/nodes`.

## Phases

### Phase 0: Schema, edge roles, and server search

- [x] Constants in `data/schema.go` and `ui/src/lib/siot/schema.ts`;
      `schema_ts_test.go` passes.
- [x] `edge_role.go`: the three types in `treeScopedTypes`, `widget` and `link`
      owned by `dashboard`.
- [x] `getNodesSearch` in `store/jetstream.go` and the `search` case in
      `nodesRequest`, with tests for scope, ranking, limit, the visit cap, and
      ancestors in the reply.
- [x] `docs/ref/api.md`: the two parameters.
- [x] `siot_test` passes. Commit.

### Phase 1: Node reference picker and rule editing

- [x] `searchNodes` in `frontend/lib/siot-nats.js` and its types;
      `connection.ts` and `tree.svelte.ts` expose `searchNodes` and
      `resolveNode` with the ID cache.
- [x] `NodeRefInput.svelte` with vitest coverage of ranking display, keyboard
      handling, and the not-found state.
- [x] `EditorProps` gains `searchNodes` and `resolveNode`; `TreePage.svelte`
      supplies them.
- [x] Condition and action editors use the picker and the point type and key
      options from the chosen node.
- [x] `docs/user/rules.md`: choosing a node. Changelog entry.
- [x] `ui/e2e/picker.spec.ts`: a browser test of the picker in a condition (run
      with `SIOT_E2E_CHANNEL=chrome` where the downloaded browser is missing).
- [x] `siot_test` passes. Commit.

### Phase 2: Node types, viewer, grid host

- [ ] `ui/src/lib/nodes/dashboard/`, `widget/`, `link/` with descriptors and
      editors; the widget editor uses the picker for bindings.
- [ ] `ui/src/lib/widgets/` registry with `value`, `gauge`, `status`, and
      `trend` (live, `TrendBuffer`); `widget` on `NodeTypeDescriptor` for
      `auto`, set on the types that have an obvious widget.
- [ ] `bindings.svelte.ts` with tests for the three states.
- [ ] `Dashboard.svelte`, `Grid.svelte`, the unknown-component placeholder, the
      one-column reflow.
- [ ] Routes `/dash` and `/dash/[id]`, `?kiosk`, header links, the Open button
      in the tree.
- [ ] Vendor the nautilus pieces this phase needs with attribution and the token
      mapping.
- [ ] `docs/user/dashboards.md` started; `docs/user/ui.md` replaces the Grafana
      paragraph with a pointer. Changelog entry.
- [ ] `siot_test` passes; `siot_build_ui` and `ui/build` staged. Commit.

### Phase 3: Edit mode, faceplates, home dashboard

- [ ] Edit toggle; drag and resize writing position points; add widget from a
      component list or a node through the picker with `auto`.
- [ ] Widget panel: the widget editor beside the grid, live.
- [ ] Faceplate on tap: dialog above 900px, `/nodes/[id]` below.
- [ ] `homeDashboard` on the user editor with the picker; `/` redirect rules;
      the Playwright smoke test covers sign-in to home dashboard.
- [ ] Docs and changelog. `siot_test` passes. Commit.

### Phase 4: Canvas host and links

- [ ] `Canvas.svelte` with scaling, pan, pinch-zoom, and the card fallback.
- [ ] Equipment widgets `tank`, `pump`, `valve` with ports; `link` rendering
      with anchoring, orthogonal routing, and the three pipe states.
- [ ] Link editing: draw from port to port, drag vertices, the link editor.
- [ ] Unit tests on the vendored `mimic.ts` helpers, carried from nautilus's
      `tests/mimic.test.ts`.
- [ ] Docs and changelog. `siot_test` passes. Commit.

### Phase 5: Relative bindings, nested dashboards, wrap-up

- [ ] `bindPath` resolver and the picker's relative toggle, with tests.
- [ ] `dashboard` widget for nesting; a cycle guard.
- [ ] `docs/ref/frontend.md`: a dashboards section; `CLAUDE.md`: the widgets
      directory and the `auto` rule; plans index. Commit.

### Follow-ups (not in this plan)

- Trends over stored points, once the history request exists.
- Kiosk playlists; a group default dashboard.
- Semantic zoom on the canvas.
- Custom components from team repos, sandboxed, per `../ui-planning.md`.
- Gridstack, if the in-house drag and resize grows past its welcome.

## Testing Strategy

- **Go:** store tests for search scope, ranking, limit, ancestors, and the visit
  cap; the edge role classification test; `schema_ts_test`.
- **Unit (vitest):** the picker, the bindings resolver states, the `auto`
  lookup, grid reflow, the mimic helpers, and the relative resolver.
- **Static:** `svelte-check`, eslint, prettier, as today.
- **Manual, each phase:** a lift station dashboard on the dev instance with a
  signal generator feeding a tank, a gauge, and a trend; a phone-width window
  for the reflow and card fallback.
- **Browser (opt-in):** the smoke test extended with the home dashboard.

## Open Questions

- Search ranking: is prefix, substring, ID, type enough, or should recent picks
  rank first?
- Gridstack or the in-house grid; decided after phase 3.
- The `bindPath` syntax: `type/description` segments, or a glob.
- Whether the `via` parent for relative bindings should be remembered per user,
  as expanded nodes are.

## Key Files

- `data/schema.go`, `data/edge_role.go`, `ui/src/lib/siot/schema.ts`: types and
  constants.
- `store/store.go` (`nodesRequest`, `handleUserRequest`), `store/jetstream.go`
  (`getNodes`, `getNodesDepth`): the search.
- `frontend/lib/siot-nats.js`, `ui/src/lib/siot/connection.ts`,
  `ui/src/lib/siot/tree.svelte.ts`: requests and the store.
- `ui/src/lib/inputs/`, `ui/src/lib/nodes/types.ts`,
  `ui/src/lib/nodes/condition/Editor.svelte`,
  `ui/src/lib/nodes/action/Editor.svelte`: the picker and its first users.
- `ui/src/lib/nodes/registry.ts`, `ui/src/lib/nodes/index.ts`: the pattern the
  widget registry copies.
- `ui/src/lib/components/Header.svelte`, `ui/src/routes/`: navigation.
- `../other-projects/nautilus/hmi/src/lib/mimic.ts`,
  `.../components/Mimic.svelte`, `.../components/Pipe.svelte`: what is vendored.
- `docs/ref/api.md`, `docs/ref/frontend.md`, `docs/user/ui.md`,
  `docs/user/rules.md`, `docs/user/dashboards.md`: documentation.
