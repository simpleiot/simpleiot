// @vitest-environment jsdom
import { describe, expect, test } from "vitest"
import { sanitizeSvg } from "./sanitize.ts"

const hostile = `<?xml version="1.0"?>
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 100 100">
  <style>
    @import url("https://evil.example/x.css");
    .glass { fill: #ddd; }
    .glass[data-siot-state="on"] { fill: #0a0; }
    .bg { background: url(https://evil.example/t.png); }
    .ok { fill: url(#grad); }
  </style>
  <script>alert(1)</script>
  <foreignObject><div onclick="alert(2)">x</div></foreignObject>
  <a href="https://evil.example"><text>link</text></a>
  <image href="https://evil.example/t.png" />
  <image href="data:image/png;base64,AAAA" />
  <use href="https://evil.example/s.svg#x" />
  <use xlink:href="#sym" />
  <circle class="glass" cx="50" cy="50" r="10" onload="alert(3)" data-siot-bind="lit" data-siot-toggle fill="url(http://evil.example/p)" />
  <rect x="1" y="1" width="2" height="2" style="fill: url(https://evil.example/q.png)" fill="url(#grad)" />
</svg>`

describe("sanitizeSvg", () => {
	const out = sanitizeSvg(hostile)

	test("removes scripts, handlers, links, and embedded documents", () => {
		expect(out).not.toContain("<script")
		expect(out).not.toContain("onload")
		expect(out).not.toContain("onclick")
		expect(out).not.toContain("<foreignObject")
		expect(out).not.toContain("<a ")
		expect(out).not.toContain("evil.example")
	})

	test("keeps the drawing, its data attributes, and local references", () => {
		expect(out).toContain("<circle")
		expect(out).toContain('data-siot-bind="lit"')
		expect(out).toContain("data-siot-toggle")
		expect(out).toContain('[data-siot-state="on"] { fill: #0a0; }')
		expect(out).toContain("fill: url(#grad)")
		expect(out).toContain('xlink:href="#sym"')
		expect(out).toContain("data:image/png;base64,AAAA")
		expect(out).toContain('fill="url(#grad)"')
	})

	test("rewrites external references in a stylesheet and drops them on elements", () => {
		expect(out).not.toContain("@import")
		expect(out).toContain("background: none")
		const doc = new DOMParser().parseFromString(out, "text/html")
		const circle = doc.querySelector("circle")
		expect(circle?.getAttribute("fill")).toBeNull()
		const rect = doc.querySelector("rect")
		expect(rect?.getAttribute("style")).toBeNull()
		expect(rect?.getAttribute("fill")).toBe("url(#grad)")
		expect(doc.querySelectorAll("image[href]").length).toBe(1)
		const uses = [...doc.querySelectorAll("use")].map(
			(u) => u.getAttribute("href") ?? u.getAttribute("xlink:href")
		)
		expect(uses).toEqual([null, "#sym"])
	})

	test("text that is not an SVG leaves nothing", () => {
		expect(sanitizeSvg("<script>alert(1)</script>")).toBe("")
		expect(sanitizeSvg("<div>hello</div>")).toBe("hello")
	})
})
