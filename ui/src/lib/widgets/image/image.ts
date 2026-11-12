// An image held in a file node's data: an SVG as text, or a PNG, JPEG,
// GIF, WebP, or SVG uploaded as binary, which the upload stores as base64.
// The kind is read from the contents rather than the file name, so a
// widget that sees only the data can draw it. The result is a data URL for
// an <img>, which loads nothing else and runs no script; an SVG is also
// sanitized, as every drawing is.
import { sanitizeSvg } from "../svg/sanitize.ts"

export interface ImageSource {
	// src is the data URL to show, "" when there is none
	src: string
	// kind is "svg" or the raster type, "" when there is no image
	kind: "" | "svg" | "png" | "jpeg" | "gif" | "webp"
	// error says why there is nothing to show
	error: string
}

const none = (error: string): ImageSource => ({ src: "", kind: "", error })

// isSvgText says whether text is SVG markup.
export function isSvgText(text: string): boolean {
	return /^\s*(<\?xml|<!--|<!doctype|<svg)/i.test(text) && /<svg[\s>]/i.test(text)
}

// raster types by the base64 of their first bytes
const rasters: readonly { kind: ImageSource["kind"]; prefix: string; mime: string }[] = [
	{ kind: "png", prefix: "iVBORw0KGgo", mime: "image/png" },
	{ kind: "jpeg", prefix: "/9j/", mime: "image/jpeg" },
	{ kind: "gif", prefix: "R0lGOD", mime: "image/gif" },
	{ kind: "webp", prefix: "UklGR", mime: "image/webp" },
]

function svgSource(text: string): ImageSource {
	const clean = sanitizeSvg(text)
	return clean === ""
		? none("the SVG has nothing left to draw once sanitized")
		: {
				src: `data:image/svg+xml;charset=utf-8,${encodeURIComponent(clean)}`,
				kind: "svg",
				error: "",
			}
}

// bytes reads base64 as a string of byte values, or undefined when it is
// not base64
function bytes(b64: string): string | undefined {
	try {
		return atob(b64)
	} catch {
		return undefined
	}
}

// decode reads base64 as UTF-8 text, or undefined when it is not base64
function decode(b64: string): string | undefined {
	const bin = bytes(b64)
	return bin === undefined
		? undefined
		: new TextDecoder().decode(Uint8Array.from(bin, (c) => c.charCodeAt(0)))
}

// svgText returns the SVG markup a file node's data holds, as text or as
// base64, or "" when it holds none.
export function svgText(data: string): string {
	if (isSvgText(data)) {
		return data
	}
	const b64 = data.replace(/\s+/g, "")
	const text = /^[A-Za-z0-9+/]+={0,2}$/.test(b64) ? decode(b64) : undefined
	return text !== undefined && isSvgText(text) ? text : ""
}

// imageSource reads a file node's data as an image.
export function imageSource(data: string): ImageSource {
	if (data.trim() === "") {
		return none("the file has no data yet")
	}
	if (isSvgText(data)) {
		return svgSource(data)
	}
	const b64 = data.replace(/\s+/g, "")
	if (!/^[A-Za-z0-9+/]+={0,2}$/.test(b64)) {
		return none("the file is not an image; upload a PNG, JPEG, GIF, WebP, or SVG")
	}
	const raster = rasters.find((r) => b64.startsWith(r.prefix))
	if (raster) {
		// a RIFF file is WebP only with WEBP at bytes 8 to 11
		if (raster.kind !== "webp" || bytes(b64.slice(0, 16))?.slice(8, 12) === "WEBP") {
			return { src: `data:${raster.mime};base64,${b64}`, kind: raster.kind, error: "" }
		}
	}
	const text = decode(b64)
	if (text !== undefined && isSvgText(text)) {
		return svgSource(text)
	}
	return none("the file is not an image; upload a PNG, JPEG, GIF, WebP, or SVG")
}
