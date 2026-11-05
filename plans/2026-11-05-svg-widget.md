# Plan: Custom SVG widget

**Branch:** `cbrake/master` **Branched from:** `ccbaf16b`

## Context

The [dashboards plan](2026-11-03-dashboards.md) shipped a fixed set of widgets:
tiles for the grid and three equipment drawings for the canvas, each a Svelte
component compiled into the UI. A team that wants a drawing of its own
equipment, a site plan, or a panel face has no way to show one. The dashboards
plan lists "custom components from team repos, sandboxed" as a follow-up; this
plan takes the smaller step that covers most of that need: a widget that draws
an SVG the user uploads, with elements inside the drawing bound to live points.

What exists today that the plan builds on:

- **File nodes** (`docs/user/file.md`) hold a file's contents in the `data`
  point, uploaded from the browser by the file editor through
  `store.uploadFile`. They sync like any node, carry `hash` and `size`, and a
  point travels in one NATS message, so a file is limited by the server's
  maximum payload (1 MB by default).
- **Widgets** register as directories under `ui/src/lib/widgets/` with a
  descriptor and a props-only component (`registry.ts`, `WidgetProps`). A canvas
  widget with `fit: "width"` takes its width as a prop and draws at that size.
  `WidgetFrame.svelte` renders it and marks a dead binding.
- **Bindings** are `bindNode`, `bindType`, `bindKey`, and `bindPath` points
  keyed by prop name. `parseBindings` in `nodes/dashboard/bindings.ts` reads
  whatever `bind*` points a widget node carries, so the runtime already handles
  binding names a descriptor does not list; only the widget editor
  (`nodes/widget/Editor.svelte`) lists bindings from the descriptor.
- **The nested dashboard widget** binds to a node rather than a point, and
  `BindingInputs.svelte` filters its picker to dashboard nodes by checking the
  binding's name.
- **Ports** come from the descriptor, overridden by `port.*` points on the
  widget (`parsePorts`).
- **The content security policy** (`api/server.go`) allows scripts only by hash
  and inline styles, so an inline script in injected markup would not run, but
  that is a second layer, not the control.

Out of scope: widgets written as code (Svelte or JavaScript) loaded at runtime,
raster images, SVGs that fetch anything, editing the drawing in the browser.

## Goals

1. An `svg` widget that draws an uploaded SVG at the widget's size, on a canvas
   or in a grid tile.
2. A small set of `data-siot-*` attributes an author adds to elements in the
   drawing to bind them to points: text, on/off state, rotation, and fill level.
3. The widget editor lists the bindings the drawing declares, and the picker for
   the drawing is filtered to file nodes.
4. Ports declared in the drawing, so links attach to it on a canvas.
5. Every uploaded SVG is sanitized before it reaches the page; a test proves
   scripts, event handlers, and external references are removed.
6. Docs, a demo drawing, changelog, and `siot_test` green at every phase.

## Design Decisions

### The drawing is a file node

The widget binds its `svg` prop to a file node's `data` point. Nothing new is
stored on the widget: the drawing lives in a file node wherever the team keeps
it, usually under the same group as the dashboard, and the binding points at it
the way the nested dashboard binding points at a dashboard. This reuses the
upload path, the size and hash points, sync, and permissions. Several widgets
can share one drawing, and a relative binding (`file/Pump drawing`) lets a
mirrored dashboard pick up a per-device drawing when that is wanted.

The file is uploaded as text (the **Binary** option off). A `data` point that
does not parse as SVG draws the widget as a placeholder naming the problem.

### Binding attributes

An author marks elements in the drawing with data attributes. Everything else in
the SVG is left as drawn, so a file from Inkscape or draw.io works once the
attributes are added in a text editor or the tool's XML editor.

| Attribute                   | On an element                                                                     |
| --------------------------- | --------------------------------------------------------------------------------- |
| `data-siot-bind="name"`     | names the binding the element follows                                             |
| `data-siot-text`            | replaces the element's text with the value                                        |
| `data-siot-decimals="1"`    | decimals for `data-siot-text`; `data-siot-unit` appends                           |
| `data-siot-range="0,100"`   | the value range `rotate` and `level` map from; 0 to 100 when absent               |
| `data-siot-rotate="-90,90"` | rotates from the first angle at the range's low end to the second at its high end |
| `data-siot-level`           | scales the element's height by the value over the range, from the bottom          |
| `data-siot-port="name,dir"` | makes the element's center a port (see below)                                     |
| `data-siot-toggle`          | a click sends the opposite of the bound value (see Controls)                      |
| `data-siot-set="1"`         | a click sends that value                                                          |

Every bound element also gets a `data-siot-state` attribute of `on`, `off`,
`stale`, or `dead` (`on` when the value is not zero), and the widget sets
nothing else. The drawing's own `<style>` does the rest: a lamp turns green with
`[data-siot-state="on"] { fill: #0a0 }`, and a dead element can hide itself.
This keeps the attribute set small and leaves styling where an SVG author
expects it.

A binding name may appear on several elements, and an element without one
follows the nearest ancestor's, so a group bound once can hold several buttons.
The widget's bindings, for the editor, are the distinct `data-siot-bind` values
in the drawing, in document order, after the `svg` binding itself.

`rotate` and `level` are set as a `transform` on the element with
`transform-box: fill-box` and the origin at the element's center or bottom
center, so the author draws the needle or the liquid at its full extent and the
widget scales it. A text value shows `text` when the point has one and the
formatted number otherwise, with the unit appended as given: the sanitizer trims
attribute values, so a space before the unit is drawn in the SVG itself.

### Rendering

The widget component:

1. Sanitizes the `svg` binding's text through DOMPurify with the SVG profile,
   plus a hook that removes `href` and `xlink:href` values that are not fragment
   references, `<style>` rules that import or reference a URL other than a
   fragment, and `<foreignObject>`. The result is cached by the text, so a value
   change does not re-sanitize.
2. Injects the sanitized markup once per drawing with `{@html}` into a container
   sized by `width`; the root `<svg>` is given `width="100%"` and
   `height="auto"`, and a root without a `viewBox` but with `width` and `height`
   gets one synthesized from them so it scales.
3. Applies the bindings in an `$effect` over `values` by querying the injected
   DOM, so a value change touches attributes and text and leaves the drawing's
   own CSS animations running.

`fit: "width"` and a default width of 200. In a grid tile the drawing fills the
tile's width and the tile's height clips it, as the other drawings do.

The descriptor also declares which binding holds the drawing and how to read
bindings and ports out of it, so the editor and the dashboard page stay generic:

```ts
// in WidgetDescriptor
source?: string                                  // the binding that holds a drawing
bindingsIn?: (text: string) => readonly string[]
portsIn?: (text: string) => readonly MimicPort[]
bindingPoints?: Readonly<Record<string, string>> // point type a binding reads by default
```

`bindingPoints` is `{ svg: "data" }`, so the binding reads the file node's
contents without the user choosing the point. A descriptor cannot name a node
type (the nodes coverage test forbids `NodeType` outside `src/lib/nodes`), so
the picker's filter to file nodes for a `source` binding lives in
`BindingInputs.svelte` beside the nested dashboard's filter. `WidgetProps` gains
`writable`, the bindings `onWrite` accepts, since a drawing has several and only
some may be written.

### Sanitizing is the control

A file node is writable by any user with write access to its group, and a
dashboard is viewed by every user in the group with their own sign-in, so an
unsanitized drawing is a stored script injection from one user to another. The
sanitizer runs on every render path, the content security policy is the second
layer, and `svg/sanitize.test.ts` is the proof: a drawing with `<script>`, an
`onload` handler, a `<foreignObject>`, an external `<image href>`, a `<use>` of
an external file, and a `<style>` with `@import` comes out with all of them
removed and its `data-siot-*` attributes and local `<style>` kept.

DOMPurify over a walker of our own because namespace confusion and mutation
cases are where a hand-written allowlist fails, and DOMPurify is maintained for
exactly that. It adds about 20 kB gzipped to the UI bundle.

### Controls

A drawing is a control the way a status pill is: `writeTargets` already picks
every binding whose node type accepts writes and whose point type is `value`,
unless the widget's `control` option is off, and the frame hands the widget
`onWrite`. The two click attributes say which element sends what:

- `data-siot-toggle` sends 1 when the bound value is 0 and 0 otherwise, so a
  lamp bound to a variable or a GPIO output is a switch.
- `data-siot-set="<value>"` sends that value, so a start and a stop button are
  two elements bound to the same name with `1` and `0`. A value that is not a
  number is sent as text.

Both act on the element's own `data-siot-bind`. The widget attaches one click
listener on its container and walks up from the event target to the nearest
element carrying either attribute, so a bulb drawn as a glass circle and a glow
circle, both bound to `lit`, toggles from either. Elements without an attribute
ignore clicks, so a drawing can mix a switch with a reading. A click sends only
when the binding is in `writes`; otherwise the attribute is inert, and the
element gets `cursor: pointer` only when it would send, so a drawing bound to a
read-only sensor shows no affordance. In edit mode clicks are swallowed, as for
the other controls, and `confirm` works unchanged since the frame handles it.

A text setting edited in place, like the value tile's edit box, is left out: an
input inside an SVG is awkward, and a value tile beside the drawing does it.

### Ports from the drawing

An element with `data-siot-port="suction,left"` declares a port at the center of
its box, computed from the element's `cx`/`cy` or `x`/`y`/`width`/`height`
attributes as fractions of the root `viewBox`, without rendering. `portsIn`
returns them, the dashboard page passes them to `parsePorts` as the defaults in
place of the descriptor's static list, and `port.*` points still override. An
element whose position cannot be read from attributes is skipped.

### Editing flow

In the widget panel, choosing the **svg** component shows the `svg` binding with
the picker filtered to file nodes. Once a file is bound, the editor resolves it,
reads `data`, and lists the drawing's bindings below, each with the usual node,
type, and key inputs. Changing the drawing changes the list; a binding the
drawing no longer names stays editable while its points exist, so nothing is
lost silently.

Uploading a drawing is done on a file node in the tree, as for any file. The
faceplate and widget panel (`NodeHost.svelte`) stub `onUploadFile`; phase 3
gives it the hidden file input the tree row has, so a file node opened from a
dashboard can take an upload too.

## Changes to note in the changelog

- A new `svg` widget; nothing changes for existing dashboards or deployments.
- `BindingInputs` filters by the descriptor rather than the binding name; no
  user-visible change.

## Phases

### Phase 1: The widget, sanitizing, and bindings

- [x] `dompurify` added to `ui/package.json`;
      `ui/src/lib/widgets/svg/sanitize.ts` with the profile and hooks, and
      `sanitize.test.ts` as above.
- [x] `ui/src/lib/widgets/svg/drawing.ts`: `parseDrawing(text)` returning the
      root `viewBox`, the binding names, the ports, and an error for text that
      is not an SVG; `applyBindings(root, values)` for the four effects and the
      state attribute; `clickTarget(root, el, writes)` returning the binding and
      the write a click sends, or nothing. Tests in jsdom for each effect, the
      state values, number formatting, the synthesized `viewBox`, the not-an-SVG
      case, toggle from on and from off, set with a number and with text, a
      click outside any attribute, and a click on a binding that is not
      writable.
- [x] `WidgetDescriptor` gains `source`, `bindingsIn`, `portsIn`, and
      `bindingTypes`; `registry.test.ts` lists the new widget.
- [x] `ui/src/lib/widgets/svg/index.ts` and `Widget.svelte`: injection once per
      drawing, the effect over values, the dead and placeholder states, the
      click listener calling `onWrite`, the pointer cursor on writable elements,
      and nothing sent while editing.
- [x] `Dashboard.svelte` passes `portsIn` ports as the defaults to `parsePorts`
      when the descriptor has a source binding.
- [x] `siot_test` passes. Commit.

### Phase 2: Editor

- [ ] `BindingInputs.svelte` filters the picker by `bindingTypes`; the nested
      dashboard widget declares its type there.
- [ ] `nodes/widget/Editor.svelte` lists the drawing's bindings after the
      declared ones when the descriptor has a source, resolving the bound file
      through `resolveNode`, and keeps bindings that still have points.
- [ ] A hint under the bindings naming the attribute set, with a link to the
      docs.
- [ ] `siot_test` passes. Commit.

### Phase 3: Upload from a dashboard, demo, docs

- [ ] `NodeHost.svelte` gets the hidden file input and `onUploadFile`, so a file
      node's editor works in the faceplate dialog and the widget panel.
- [ ] `testdata/custom-widget-demo.yaml`: a dashboard that shows every feature
      of the widget, loaded with `siot import` like the dashboard demo. Under
      one group:
  - **Drawings as file nodes**, each written inline as a YAML block scalar so
    the file reads as a worked example: a lamp (`state` and `toggle`), a dial
    with a needle (`rotate`, with `text`, `decimals`, and `unit` in the face), a
    vessel (`level` with a port on each side), a start and stop panel (two
    elements bound to one name with `set` 1 and 0), and a pump housing whose
    `port` elements are declared in the drawing.
  - **Signals and switches** that drive them: a signal generator for the dial
    and the vessel, an on/off variable the lamp toggles, and a variable the
    start and stop buttons set, with a rule that runs the pump signal from it.
  - **A canvas dashboard** placing every drawing with a link from the pump's
    discharge port to the vessel's inlet port, and a `port.*` override on one
    widget to show the override winning over the drawing.
  - **A grid dashboard** showing the lamp and the dial as tiles, so both hosts
    are covered, and one widget with `control` off bound to the same variable,
    to show an inert toggle.
  - **The three binding states**: one lamp bound to a point no node writes yet
    (stale) and one bound to a node description that does not exist (dead), so
    the dashed frame and the state attribute can be seen.
  - **A relative binding**: the lamp dashboard mirrored under two devices, each
    with its own `file/Lamp drawing` and variable, so the mirrored dashboard
    draws each device's own drawing.
  - **An Overview** of the two as nested dashboards. A comment at the top of the
    file lists the features and where each one is.
- [ ] `ui/e2e/svg.spec.ts`: a browser test that imports the demo, opens the
      canvas, checks the lamp's state attribute follows its variable, clicks the
      lamp to flip it, and checks the link attaches to the vessel's port.
- [ ] `docs/user/dashboards.md`: a **Custom drawings** section with the
      attribute table, the sanitizing note, and a paragraph on the demo file;
      `docs/user/file.md` points at it; `docs/ref/frontend.md` names the
      sanitizer. Changelog entry. `CLAUDE.md` notes the descriptor's source
      hooks beside the widget registry note.
- [ ] `siot_test` passes. Commit.

## Testing Strategy

- **Unit (vitest):** sanitizer removals and keeps; drawing parse (bindings,
  ports from `circle` and `rect`, `viewBox` synthesis, not-SVG); effects in
  jsdom (text with decimals and unit, state for on, off, stale, dead, rotate and
  level transforms); click targets (toggle, set, inert, not writable); registry
  listing.
- **Browser (Playwright):** the demo drawing on a canvas, a state change
  reaching the DOM, a click on the lamp flipping the switch, and a link attached
  to a drawing's port.
- **Manual:** the demo file loaded into a scratch instance, every feature
  checked against the list in its header comment, in light and dark themes, on a
  phone width; an Inkscape export and a draw.io export with attributes added
  through the tools' XML editors.

## Open Questions

- Whether `data-siot-level` should scale height or clip with a mask. Scaling
  distorts a rounded liquid outline; a mask keeps the shape. Start with the
  transform, since it needs no extra elements, and revisit if the demo looks
  wrong.
- A `fill` mapping for a color ramp (temperature tinting as the tank does). Left
  out until asked; the state attribute and CSS cover on/off.

## Follow-ups (not in this plan)

- An `image` widget for a raster logo or photo through the same file binding.
- Custom components from team repos, as the dashboards plan lists.
- Bindings in the drawing to `prop` options, so one drawing can be reused with
  different fixed labels.
