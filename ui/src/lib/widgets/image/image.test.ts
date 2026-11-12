// @vitest-environment jsdom
import { describe, expect, test } from "vitest"
import { imageSource, isSvgText, svgText } from "./image.ts"

const b64 = (s: string) => btoa(s)
const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect width="10" height="10"/><script>alert(1)</script></svg>`

describe("image source", () => {
	test("SVG text is sanitized into a data URL", () => {
		const s = imageSource(svg)
		expect(s.kind).toBe("svg")
		expect(s.src.startsWith("data:image/svg+xml;charset=utf-8,")).toBe(true)
		expect(decodeURIComponent(s.src)).not.toContain("script")
	})

	test("an SVG uploaded as binary is read from its base64", () => {
		expect(imageSource(b64(svg)).kind).toBe("svg")
		expect(svgText(b64(svg))).toBe(svg)
		expect(svgText(svg)).toBe(svg)
		expect(svgText(b64("GIF89a0000"))).toBe("")
	})

	test("raster types are found by their first bytes", () => {
		expect(imageSource(b64("\x89PNG\r\n\x1a\n0000")).kind).toBe("png")
		expect(imageSource(b64("\xff\xd8\xff\xe00000")).kind).toBe("jpeg")
		expect(imageSource(b64("GIF89a0000")).kind).toBe("gif")
		expect(imageSource(b64("RIFF\xff\xff\x00\x00WEBPVP8 ")).kind).toBe("webp")
		expect(imageSource(b64("RIFF\x00\x00\x00\x00WAVEfmt ")).kind).toBe("")
		const png = imageSource(b64("\x89PNG\r\n\x1a\n0000"))
		expect(png.src.startsWith("data:image/png;base64,iVBORw0KGgo")).toBe(true)
	})

	test("anything else says why there is nothing to show", () => {
		expect(imageSource("").error).toContain("no data")
		expect(imageSource("hello, world").error).toContain("not an image")
		expect(imageSource(b64("plain text")).error).toContain("not an image")
	})

	test("SVG text is recognized with a prolog or a comment first", () => {
		expect(isSvgText(`<?xml version="1.0"?>\n${svg}`)).toBe(true)
		expect(isSvgText(`<!-- a lamp -->${svg}`)).toBe(true)
		expect(isSvgText("<html><svg></svg></html>")).toBe(false)
	})
})
