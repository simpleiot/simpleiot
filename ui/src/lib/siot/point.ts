// Points and the helpers that read and update lists of them, ported from
// the Elm UI's Api/Point.elm.

import { PointType } from "./schema.ts"

// Point mirrors data.Point in the shape the codec in simpleiot-js reads
// and writes. time is an ISO 8601 string; value is 0 for text points and
// text is "" for numeric ones.
export interface Point {
	type: string
	key: string
	time: string
	dataType: number
	value: number
	text: string
	tombstone: number
	origin?: string
}

// DataType is the wire data type, as in data/point.go.
export const DataType = {
	Unknown: 0,
	Float: 1,
	Int: 2,
	String: 3,
	JSON: 4,
} as const

// blankMagicValue is the text a number input holds while it is empty, so
// the edit is kept until saved; clearText drops it.
export const blankMagicValue = "123BLANK123"

export const epoch = new Date(0).toISOString()

export function newText(type: string, key: string, text: string, time = epoch): Point {
	return { type, key, time, dataType: DataType.String, value: 0, text, tombstone: 0 }
}

export function newValue(type: string, key: string, value: number, time = epoch): Point {
	return { type, key, time, dataType: DataType.Float, value, text: "", tombstone: 0 }
}

// newTombstone deletes the point with the type and key.
export function newTombstone(type: string, key: string, time = epoch): Point {
	return { type, key, time, dataType: DataType.Unknown, value: 0, text: "", tombstone: 1 }
}

const normKey = (key: string) => (key === "" ? "0" : key)

export function get(points: readonly Point[], type: string, key = "0"): Point | undefined {
	const k = normKey(key)
	return points.find((p) => p.type === type && p.key === k)
}

export function getAll(points: readonly Point[], type: string): Point[] {
	return points.filter((p) => p.type === type)
}

export function getText(points: readonly Point[], type: string, key = "0"): string {
	return get(points, type, key)?.text ?? ""
}

export function getValue(points: readonly Point[], type: string, key = "0"): number {
	return get(points, type, key)?.value ?? 0
}

export function getBool(points: readonly Point[], type: string, key = "0"): boolean {
	return getValue(points, type, key) === 1
}

const isInt = (s: string) => /^[+-]?\d+$/.test(s)

// keyInt reads a key as an integer, 0 when it is not one.
const keyInt = (key: string) => (isInt(key) ? Number.parseInt(key, 10) : 0)

// getTextArray returns the text of the live points of a type in key
// order, for points that hold a list with one entry per key.
export function getTextArray(points: readonly Point[], type: string): string[] {
	return points
		.filter((p) => p.type === type && p.tombstone === 0)
		.reverse()
		.sort((a, b) => keyInt(a.key) - keyInt(b.key))
		.map((p) => p.text)
}

// getBestDesc names a node: a user's name, else its description.
export function getBestDesc(points: readonly Point[]): string {
	const first = getText(points, PointType.FirstName)
	if (first !== "") {
		return `${first} ${getText(points, PointType.LastName)}`
	}
	const desc = getText(points, PointType.Description)
	return desc === "" ? "no description" : desc
}

export function getLatest(points: readonly Point[]): Point | undefined {
	let latest: Point | undefined
	for (const p of points) {
		if (!latest || Date.parse(p.time) > Date.parse(latest.time)) {
			latest = p
		}
	}
	return latest
}

// updatePoint replaces the point with the same type and key, or adds it.
export function updatePoint(points: readonly Point[], point: Point): Point[] {
	const i = points.findIndex((p) => p.type === point.type && p.key === point.key)
	if (i >= 0) {
		const out = points.slice()
		out[i] = point
		return out
	}
	return [point, ...points]
}

// updatePoints applies edits to a list of points, the last edit to a type
// and key winning.
export function updatePoints(points: readonly Point[], newPoints: readonly Point[]): Point[] {
	let out = points.slice()
	for (let i = newPoints.length - 1; i >= 0; i--) {
		out = updatePoint(out, newPoints[i] as Point)
	}
	return out
}

// mergePoints applies points that arrived from the server, keeping the
// newer of two points with the same type and key, so a point delivered
// late does not replace a newer one.
export function mergePoints(points: readonly Point[], newPoints: readonly Point[]): Point[] {
	let out = points.slice()
	for (let i = newPoints.length - 1; i >= 0; i--) {
		const p = newPoints[i] as Point
		const cur = get(out, p.type, p.key)
		if (cur && Date.parse(p.time) < Date.parse(cur.time)) {
			continue
		}
		out = updatePoint(out, p)
	}
	return out
}

// clearText drops the text a number input keeps while it is being edited,
// so a number point is sent as a number.
export function clearText(points: readonly Point[]): Point[] {
	return points.map((p) => (p.value !== 0 || p.text === blankMagicValue ? { ...p, text: "" } : p))
}

export function filterDeleted(points: readonly Point[]): Point[] {
	return points.filter((p) => p.tombstone === 0)
}

// compare orders points by type, then key (numerically when both are
// integers), then value.
export function compare(a: Point, b: Point): number {
	if (a.type !== b.type) {
		return a.type < b.type ? -1 : 1
	}
	if (isInt(a.key) && isInt(b.key)) {
		const d = keyInt(a.key) - keyInt(b.key)
		if (d !== 0) {
			return d
		}
	}
	if (a.key !== b.key) {
		return a.key < b.key ? -1 : 1
	}
	return a.value - b.value
}

// sorted returns the points in compare order.
export function sorted(points: readonly Point[]): Point[] {
	return points.slice().sort(compare)
}
