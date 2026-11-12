// Test values for a drawing shown in the file editor. Nothing is sent: a
// value typed here, or set by clicking the drawing, only exercises it.
import type { Bound, Write } from "#lib/widgets/registry.ts"

// testBound reads a typed test value as a bound value: stale when empty, a
// number when it parses as one, and text otherwise.
export function testBound(text: string): Bound {
	const t = text.trim()
	if (t === "") {
		return { state: "stale", value: 0, text: "", time: "" }
	}
	const n = Number(t)
	return Number.isFinite(n)
		? { state: "live", value: n, text: "", time: "" }
		: { state: "live", value: 0, text: t, time: "" }
}

// writeText is what a click on the drawing puts in the test value.
export function writeText(w: Write): string {
	return w.text !== "" ? w.text : String(w.value)
}
