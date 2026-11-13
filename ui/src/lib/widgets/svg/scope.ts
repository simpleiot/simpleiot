// scope keeps a drawing's stylesheet to the drawing. A drawing is injected
// into the page, and a <style> inside an inline SVG applies to the whole
// document, so without this a rule in one drawing restyles every other
// drawing and widget on the dashboard. The rules go inside an @scope block
// rooted at the element that holds the drawing, which carries the scope
// attribute. @scope reads a selector as a descendant of its root, so the
// root sits outside the <svg> for rules that name the <svg> itself.
// @keyframes and @font-face are not allowed there, so they stay at the top
// level, and each keyframes name gets the drawing's id so that two drawings
// may both define a "spin" without one replacing the other.

export const scopeAttr = "data-siot-drawing"

// scopeId names a drawing by a hash of its text: the same drawing shown
// twice shares the id, which is harmless since the rules are the same.
export function scopeId(text: string): string {
	let h = 0x811c9dc5
	for (let i = 0; i < text.length; i++) {
		h ^= text.charCodeAt(i)
		h = Math.imul(h, 0x01000193)
	}
	return (h >>> 0).toString(36)
}

// blocks splits a stylesheet into its top-level statements: rules,
// at-rules with a block, and at-rules ending in a semicolon. Comments are
// dropped, and braces inside strings do not count.
function blocks(css: string): string[] {
	const text = css.replace(/\/\*[\s\S]*?\*\//g, "")
	const out: string[] = []
	let depth = 0
	let quote = ""
	let start = 0
	for (let i = 0; i < text.length; i++) {
		const c = text[i]
		if (quote !== "") {
			if (c === "\\") {
				i++
			} else if (c === quote) {
				quote = ""
			}
			continue
		}
		if (c === '"' || c === "'") {
			quote = c
		} else if (c === "{") {
			depth++
		} else if (c === "}") {
			depth--
			if (depth === 0) {
				out.push(text.slice(start, i + 1).trim())
				start = i + 1
			}
		} else if (c === ";" && depth === 0) {
			out.push(text.slice(start, i + 1).trim())
			start = i + 1
		}
	}
	const rest = text.slice(start).trim()
	if (rest !== "") {
		out.push(rest)
	}
	return out.filter((b) => b !== "")
}

const keyframes = /^@(-webkit-)?keyframes\s+(-?[_a-zA-Z][\w-]*)/
const global = /^@(-webkit-)?keyframes\b|^@font-face\b/i
const animation = /(animation(?:-name)?\s*:)([^;}]*)/gi

const escapeRe = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")

// scopeStyles rewrites a drawing's stylesheet so its rules apply only
// inside the element carrying scopeAttr with the given id.
export function scopeStyles(css: string, id: string): string {
	const parts = blocks(css)
	const names = new Map<string, string>()
	for (const b of parts) {
		const name = keyframes.exec(b)?.[2]
		if (name !== undefined) {
			names.set(name, `${name}-${id}`)
		}
	}
	const rename = (s: string) => {
		let out = s
		for (const [from, to] of names) {
			out = out.replace(new RegExp(`(^|[^\\w-])${escapeRe(from)}(?![\\w-])`, "g"), `$1${to}`)
		}
		return out
	}
	const top: string[] = []
	const scoped: string[] = []
	for (const b of parts) {
		const name = keyframes.exec(b)?.[2]
		if (name !== undefined) {
			top.push(b.replace(keyframes, `@$1keyframes ${names.get(name) ?? name}`))
		} else if (global.test(b)) {
			top.push(b)
		} else {
			scoped.push(
				names.size > 0
					? b.replace(animation, (_: string, p: string, v: string) => p + rename(v))
					: b
			)
		}
	}
	if (scoped.length > 0) {
		top.push(`@scope ([${scopeAttr}="${id}"]) {\n${scoped.join("\n")}\n}`)
	}
	return top.join("\n")
}
