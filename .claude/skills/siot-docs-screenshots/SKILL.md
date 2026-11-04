---
name: siot-docs-screenshots
description:
  Use when the Simple IoT documentation screenshots need refreshing, a doc page
  needs a new screenshot of the web UI, or a UI change makes existing
  screenshots stale. Triggers on requests like "update the docs screenshots",
  "take a fresh screenshot of the modbus node for the docs", "add a screenshot
  to the gpio page", "which docs pages have no screenshot", or after a change to
  the Svelte UI that alters how a node or page looks. Also use when adding a new
  node type's documentation, since the page should get a screenshot.
---

# Documentation screenshots

The screenshots in `docs/` are captured from the Svelte UI by a script, not by
hand, so a refresh after a UI change is one command and the images stay
consistent with each other. Everything lives in `scripts/` next to this file:

| File                | Role                                                                                                                                                                    |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `capture.sh`        | Builds siot, starts a scratch instance in a temp dir, imports the demo tree, runs the capture, stops everything. The only command most refreshes need.                  |
| `demo.yaml`         | The node tree the screenshots show, imported with `siot import` after `testdata/dashboard-demo.yaml`, which supplies the Overview dashboard.                            |
| `provisioning.yaml` | One provisioning file, so the provisioning node has a file to show.                                                                                                     |
| `shots.mjs`         | Playwright script. Each screenshot is a `shot(name, file, fn)` entry that names where the image lands under `docs/`, so the mapping from image to page is in one place. |
| `audit.py`          | Lists pages with no image, referenced images nothing regenerates, and generated images nothing references. Run it first and last.                                       |

## Refreshing everything

```sh
.claude/skills/siot-docs-screenshots/scripts/capture.sh
```

It needs Chrome installed (Playwright drives the system Chrome, since the
Chromium Playwright downloads does not match the installed version) and the UI
dependencies from `siot_setup`. The run takes about three minutes and ends with
`git status` of the changed images. Pass a regex to capture only matching shots
while iterating on one: `capture.sh 'modbus'`.

Review the result before committing. Build a contact sheet and look at it,
rather than opening forty files one by one:

```sh
montage docs/user/images/{modbus,rules,gps}.png -label '%f' -tile 3x \
  -geometry 600x600+6+6 /tmp/review.png
```

Things that mark a bad capture: a row showing `(not connected)` or an error
where the old image had values, a form field that is empty because the point was
given the wrong type, a header that repeats a file name, and a clipped tree
whose last row is cut off. Fix the cause in `demo.yaml` or `shots.mjs` and
recapture that shot, rather than retouching an image.

Commit the images with the doc change they belong to. Screenshot refreshes do
not need a changelog entry.

## Adding a screenshot where one is missing

Run `audit.py` to see which pages under `docs/user` and `docs/ref` have no
image. Pages about a client or node type should show the node's editor; pages
about a workflow (add, paste, raw view) show the panel involved. Pages that are
pure reference, history, or policy (the FAQ, status, installation) do not need
one, so use judgment rather than filling every gap.

For a new screenshot:

1. **Put the node in `demo.yaml`** if the tree does not have it. Give it a
   unique description, because the script and rule links both find nodes by
   description. Read the comment at the top of the file for what has to be
   static and what runs live, and look at the type's `Editor.svelte` in
   `ui/src/lib/nodes/<type>/` for the point names it reads.
2. **Add a `shot(...)` entry in `shots.mjs`**, next to the other shots for that
   page. The helpers cover the common cases: `detail` for a node with its editor
   open, `range` for a slice of the tree, `parentAndChild` for a node and one
   child both open, and `underSBC`/`underSite` for nodes that sit inside the
   example device or group. Every shot expands what it needs and collapses it
   again, so shots stay independent and can run alone.
3. **Capture just that shot** with `capture.sh '<name>'` and look at the image.
4. **Reference it from the page** with a plain Markdown image and alt text that
   says what the picture shows, placed after the paragraph that introduces the
   thing: `![an mqtt node with a subscription](images/mqtt.png)`. Use `images/`
   next to the page; user pages use `docs/user/images`, reference pages
   `docs/ref/images`, and the CAN page keeps its images in `docs/images`.
5. **Run `audit.py` again.** It should list the page as having an image and the
   shot as referenced, and it fails if any referenced image is missing.

## What stays outside the script

Some images are not UI captures and are listed in `KEEP` in `audit.py`: Grafana
views, hardware photos, the wiring schematic, an annotated flow drawn over an
old screenshot, and the historical image in ADR-7. Replace those by hand when
they go stale, and add any new hand-made image to `KEEP` so the audit does not
flag it. Diagrams are hand-written SVG files and are not screenshots; see
`docs/ref/diagrams.md`.

## How the capture works, for when something breaks

The script signs in as `admin`/`admin`, opens `/nodes`, and finds each tree row
by its accessible name, `"<Type label>: <description>"`. A row's detail is the
`:scope > .row` element of the `li[role=treeitem]`, which leaves the children
out of the image. Add and paste panels are the `.op` element inside that row,
and a collapsible point table opens with `button.toggle`. Captures are at 2x
device scale, 960 px wide, light theme, with the browser in `America/New_York`
so schedule times read the same every run.

Facts about the instance that shaped the demo tree:

- Clients run only on nodes under the root device. Nodes under the `SBC` device
  never run, so they carry static `value` points instead of live ones. That is
  how Modbus RTU, 1-wire, GPIO, CAN, serial, and Shelly get readings on a
  machine without the hardware.
- A client Modbus IO shows `(cmd pending)` until `valueSet` equals `value`.
- Schedule times are stored in UTC and shown in the browser's zone.
- A file node's header appends the file name, so the description leaves it out.
- The codec in `js/simpleiot-js/codec.js` reads integer points unsigned, so a
  negative number in the YAML is written as a float (`-5.0`) until that is
  fixed.
- A fresh browser session starts with the root collapsed; `signIn` expands it.
- The provisioning node has no description, so its shot finds it by type.
- Secret points (auth tokens, SIDs, passwords) are blank in the UI by design.

When a shot fails, the script logs `FAIL <name>` with the first line of the
error, reloads the tree, and carries on, so one broken shot does not hide the
others. The usual causes are a renamed description, a renamed button label, or a
node the import rejected (the import prints `error:` lines for a description
that matches two nodes or a point that does not decode).
