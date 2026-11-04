---
name: siot-docs-diagrams
description:
  Use when adding, changing, or reviewing a diagram in the Simple IoT
  documentation, ADRs, or plans: an architecture picture, a data flow, a node
  tree, a sequence of sync or enrollment steps, or any figure that is drawn
  rather than captured from the UI. Triggers on requests like "add a diagram
  showing X", "draw the flow for Y", "update the sync diagram", "this doc needs
  a picture", or when a design change makes an existing SVG wrong. Do not use
  for screenshots of the web UI, which siot-docs-screenshots covers.
---

# Documentation diagrams

Diagrams in this repository are plain SVG files written by hand, with one shared
style so every figure follows the reader's light or dark theme and the colors
mean the same thing on every page. The authority is
[docs/ref/diagrams.md](../../../docs/ref/diagrams.md): read its style guide
before drawing, and add the finished diagram to the gallery at the bottom of
that page. This skill is the working checklist around it, not a replacement.

Do not reach for draw.io, Mermaid, PlantUML, or a generated PNG here. The global
preference for draw.io does not apply to this repository; hand-written SVG is
the convention because the files are diffable, themeable, and need no export
step.

## Drawing a new diagram

1. **Decide what the picture has to show** in one sentence, and keep it to that.
   A reader glances at a diagram to decide whether to read on, so leave out
   details the text carries. Prefer a drawing of the mechanism (boxes for
   things, arrows for what moves between them) over a list of labels.
2. **Copy the closest existing diagram** from `docs/ref/images/`,
   `docs/user/images/`, or `docs/adr/assets/` and edit it. Every file starts
   with the same `<style>` block, `<defs>` with the arrow markers, and a
   background `<rect class="bg">`. Keep that header unchanged; if the style
   itself needs a change, make it in every file.
3. **Follow the style guide's tables** for shapes, colors, lines, and text. The
   short version: `blue` is SIOT software, `green` nodes, `yellow` points and
   data, `purple` NATS and external services, `orange` hardware, `red` a warning
   or something that does not happen, `gray` a container. Container titles start
   with an emoji that names the place (☁️ cloud, 📟 edge device, 🏭 site, 📦
   instance). Flow arrows use `marker-end="url(#arrow)"`, tree links are elbows
   with no arrowheads, dashed means logical or not running.
4. **Size and describe it.** `viewBox`, `width`, and `height` match, about 900 ×
   500 or smaller. Put a description in the `aria-label` on `<svg>`, and no
   title inside the picture, because the heading or sentence around it says what
   it shows.
5. **Place the file** in `images/` next to the page that uses it (ADRs use
   `assets/`), and reference it with a Markdown image whose alt text describes
   the picture:
   `![a device enrolls with a token, then syncs with its own key](images/sync-enrollment.svg)`.
6. **Check both themes.** Open it in a browser and switch the system theme, or
   render both with headless Chromium as the style guide shows, and look at the
   output. Text that overflows a box, an arrow that ends short of its target,
   and a label that is unreadable on the dark tint are the usual problems.
7. **Add it to the gallery** in `docs/ref/diagrams.md` under its own heading,
   with a line saying which pages use it. The gallery is how the next person
   finds the closest diagram to copy.

## Changing an existing diagram

Find every page that uses it (the gallery entry lists them, and `rg` on the file
name confirms) and make sure the change reads correctly in each context. A
diagram shared by three pages describes all three, so a detail added for one
page can mislead the others; if the pages have diverged, give one of them its
own file instead.

Keep the header block untouched and check both themes again after the edit.

## Reviewing a diagram

Judge it against the sentence it was meant to show and against the style guide's
tables. Flag a color used for decoration rather than meaning, a title inside the
picture, a missing `aria-label`, a hard-coded fill that will not follow the dark
theme, a file that is not in the gallery, and a diagram that tries to carry the
whole page's detail. Suggest what to remove before what to add.
