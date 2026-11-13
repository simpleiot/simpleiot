import { describe, expect, test } from "vitest"
import { scopeId, scopeStyles } from "./scope.ts"

describe("scope", () => {
	test("rules go inside an @scope rooted at the drawing", () => {
		const out = scopeStyles(".label { text-anchor: middle; }\ntext { fill: red }", "a1")
		expect(out).toBe(
			'@scope ([data-siot-drawing="a1"]) {\n.label { text-anchor: middle; }\ntext { fill: red }\n}'
		)
	})

	test("media queries stay inside the scope, comments are dropped", () => {
		const out = scopeStyles("/* a } brace */ @media (max-width: 9px) { .x { opacity: 0 } }", "a1")
		expect(out).toBe(
			'@scope ([data-siot-drawing="a1"]) {\n@media (max-width: 9px) { .x { opacity: 0 } }\n}'
		)
	})

	test("keyframes stay global under a name of their own", () => {
		const css = [
			".imp[data-siot-state='on'] { animation: spin 1.2s linear infinite; }",
			"@keyframes spin { to { transform: rotate(360deg); } }",
			".other { animation-name: spinner; }",
		].join("\n")
		const out = scopeStyles(css, "b2")
		expect(out).toContain("@keyframes spin-b2 { to { transform: rotate(360deg); } }")
		expect(out).toContain("animation: spin-b2 1.2s linear infinite;")
		// a longer name that starts the same is left alone
		expect(out).toContain("animation-name: spinner;")
		expect(out.indexOf("@keyframes")).toBeLessThan(out.indexOf("@scope"))
	})

	test("a brace in a string does not end a rule", () => {
		const out = scopeStyles('.x::after { content: "}"; } .y { fill: red }', "c3")
		expect(out).toBe(
			'@scope ([data-siot-drawing="c3"]) {\n.x::after { content: "}"; }\n.y { fill: red }\n}'
		)
	})

	test("the id follows the text", () => {
		expect(scopeId("<svg/>")).toBe(scopeId("<svg/>"))
		expect(scopeId("<svg/>")).not.toBe(scopeId("<svg />"))
		expect(scopeId("<svg/>")).toMatch(/^[0-9a-z]+$/)
	})
})
