#!/usr/bin/env python3
"""Reports where the documentation screenshots stand.

Run from anywhere inside the repo:

    python3 audit.py

It prints three lists:

- pages under docs/user and docs/ref with no image at all, which are the
  candidates for a new screenshot;
- raster images the docs reference that shots.mjs does not produce and
  that are not in KEEP below, which are screenshots that will go stale
  because nothing regenerates them;
- shots that shots.mjs produces but no page references, which are dead
  weight to remove.

Images in KEEP are hand-made or come from other tools (Grafana, photos,
annotated diagrams) and are expected to be outside the script.
"""

import os
import re
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.abspath(os.path.join(HERE, "..", "..", "..", ".."))
DOCS = os.path.join(ROOT, "docs")

# docs-relative paths of images that are not screenshots of the UI
KEEP = {
    "user/images/msg-process.png",  # annotated flow over an old screenshot
    "user/images/onewire-schematic.png",
    "user/images/ds18b20-photo.png",
    "user/images/gw.jpg",
    "user/images/node-tmp.jpg",
    "user/assets/image-20231108165150571.png",  # Grafana
    "user/assets/image-20240319110846431.png",  # Grafana
    "user/assets/image-20240319111845396.png",  # Grafana
    "user/assets/image-20240319112206573.png",  # Grafana
    "user/assets/gps-grafana-vm-queries.png",
    "user/assets/gps-grafana-vm-transformations.png",
    "adr/assets/nodes.png",  # historical, part of the ADR's record
    "images/simple-iot-nats-connects.jpg",
}

IMG = re.compile(r'!\[[^\]]*\]\(([^)\s]+)\)|<img[^>]*src="([^"]+)"')
RASTER = (".png", ".jpg", ".jpeg", ".gif", ".webp")


def pages():
    for sub in ("user", "ref"):
        d = os.path.join(DOCS, sub)
        for name in sorted(os.listdir(d)):
            if name.endswith(".md") and name != "SUMMARY.md":
                yield os.path.join(d, name)


def references():
    """Yields (page, docs-relative image path) for every raster image."""
    for root, _, files in os.walk(DOCS):
        for name in files:
            if not name.endswith(".md"):
                continue
            page = os.path.join(root, name)
            text = open(page, encoding="utf-8").read()
            for m in IMG.finditer(text):
                target = m.group(1) or m.group(2)
                if target.startswith("http") or not target.lower().endswith(RASTER):
                    continue
                full = os.path.normpath(os.path.join(root, target))
                yield page, os.path.relpath(full, DOCS)


def generated():
    out = subprocess.run(
        ["node", os.path.join(HERE, "shots.mjs"), "--list"],
        capture_output=True,
        text=True,
        check=True,
    ).stdout
    return {line.split("\t")[1] for line in out.splitlines() if "\t" in line}


def main():
    gen = generated()
    refs = list(references())
    referenced = {img for _, img in refs}
    rel = lambda p: os.path.relpath(p, ROOT)

    print("Pages with no image (candidates for a screenshot):")
    none = True
    for page in pages():
        text = open(page, encoding="utf-8").read()
        if not IMG.search(text):
            print("  " + rel(page))
            none = False
    if none:
        print("  (none)")

    print("\nReferenced images that nothing regenerates (not in KEEP):")
    stale = sorted({img for _, img in refs if img not in gen and img not in KEEP})
    for img in stale:
        where = sorted({rel(p) for p, i in refs if i == img})
        print(f"  {img}  <- {', '.join(where)}")
    if not stale:
        print("  (none)")

    print("\nGenerated shots that no page references:")
    unused = sorted(gen - referenced)
    for img in unused:
        print("  " + img)
    if not unused:
        print("  (none)")

    missing = sorted(img for img in referenced if not os.path.exists(os.path.join(DOCS, img)))
    if missing:
        print("\nReferenced images missing from disk:")
        for img in missing:
            print("  " + img)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
