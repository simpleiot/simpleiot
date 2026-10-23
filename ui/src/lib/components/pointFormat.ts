// Formatting for the tables of points that the metrics, serial device, and
// Shelly IO editors show below their inputs, ported from the formatters
// those Elm editors kept.

import { toDateTimeString } from "#lib/util/iso8601.ts"
import { renderPoint2, type Point } from "#lib/siot/point.ts"

export { filterSpecialPoints } from "#lib/siot/point.ts"

export interface Row {
	desc: string
	value: string
}

export interface Format {
	desc: (p: Point) => string
	vf: (p: Point) => string
}

// round formats a number with a fixed number of decimal places.
export const round = (places: number, v: number) => v.toFixed(places)

const us = (places: number, v: number) =>
	v.toLocaleString("en-US", { minimumFractionDigits: places, maximumFractionDigits: places })

export const toMiB = (p: Point) => us(1, p.value / (1024 * 1024))
export const toPercent = (p: Point) => `${round(1, p.value)} %`
export const toWhole = (p: Point) => us(0, p.value)
export const toText = (p: Point) => (p.text === "" ? " " : p.text)
// toTime reads the value as Unix seconds
export const toTime = (p: Point) => toDateTimeString(Math.round(p.value) * 1000)
export const descS = (d: string) => () => d
export const descKey = (d: string) => (p: Point) => `${d} ${p.key}`

export function formatPoint(formats: Readonly<Record<string, Format>>, p: Point): Row {
	const f = formats[p.type]
	return f ? { desc: f.desc(p), value: f.vf(p) } : renderPoint2(p)
}
