import { describe, expect, test } from "vitest"
import {
	blankMagicValue,
	clearText,
	DataType,
	formatRaw,
	getBestDesc,
	getTextArray,
	mergePoints,
	parseRaw,
	newText,
	sorted,
	updatePoints,
	type Point,
} from "./point.ts"

const t0 = new Date(0).toISOString()
const pt = (type: string, key: string, text: string, tombstone = 0): Point => ({
	type,
	key,
	time: t0,
	dataType: DataType.String,
	value: 0,
	text,
	tombstone,
})

describe("point", () => {
	test("getTextArray returns live entries in key order", () => {
		const pts = [
			pt("a", "0", "111"),
			pt("b", "0", "444"),
			pt("a", "2", "333"),
			pt("a", "1", "222"),
			pt("a", "3", "555", 1),
			pt("a", "10", "444"),
		]
		expect(getTextArray(pts, "a")).toEqual(["111", "222", "333", "444"])
	})

	test("mergePoints keeps the newer point", () => {
		const created: Point = {
			...pt("tombstone", "0", ""),
			time: new Date(1000).toISOString(),
			dataType: 1,
		}
		const deleted: Point = { ...created, time: new Date(2000).toISOString(), value: 1 }
		expect(mergePoints(mergePoints([created], [deleted]), [created])).toEqual([deleted])
	})

	test("updatePoints replaces by type and key, and adds new ones", () => {
		const out = updatePoints([pt("a", "0", "x")], [pt("a", "0", "y"), pt("b", "0", "z")])
		expect(out).toHaveLength(2)
		expect(out.find((p) => p.type === "a")?.text).toBe("y")
	})

	test("clearText drops the edit text of number points", () => {
		const num: Point = { ...pt("v", "0", "1.50"), value: 1.5 }
		const blank: Point = pt("v", "1", blankMagicValue)
		const text = pt("d", "0", "hello")
		expect(clearText([num, blank, text]).map((p) => p.text)).toEqual(["", "", "hello"])
	})

	test("getBestDesc prefers a user's name", () => {
		expect(getBestDesc([newText("firstName", "0", "Ada"), newText("lastName", "0", "L")])).toBe(
			"Ada L"
		)
		expect(getBestDesc([newText("description", "0", "pump")])).toBe("pump")
		expect(getBestDesc([])).toBe("no description")
	})

	test("sorted orders integer keys numerically", () => {
		const out = sorted([pt("a", "10", ""), pt("a", "2", ""), pt("a", "x", ""), pt("0", "0", "")])
		expect(out.map((p) => `${p.type}.${p.key}`)).toEqual(["0.0", "a.2", "a.10", "a.x"])
	})

	test("formatRaw quotes strings and leaves numbers bare", () => {
		const num = (dataType: number, value: number): Point => ({
			...pt("v", "0", ""),
			dataType,
			value,
		})
		expect(formatRaw(pt("d", "0", 'say "hi"'))).toBe('"say \\"hi\\""')
		expect(formatRaw(num(DataType.Float, 2.5))).toBe("2.5")
		expect(formatRaw(num(DataType.Int, 5))).toBe("5")
		expect(formatRaw(num(DataType.Unknown, 0))).toBe("")
	})

	test("parseRaw reads strings and numbers and keeps a number's type", () => {
		const parse = (s: string, prev?: Point) => parseRaw(s, prev, "v", "0", t0)
		const float: Point = { ...pt("v", "0", ""), dataType: DataType.Float, value: 5 }
		const int: Point = { ...float, dataType: DataType.Int }
		expect(parse('"12"')).toMatchObject({ dataType: DataType.String, text: "12", value: 0 })
		expect(parse('"a\\"b"')?.text).toBe('a"b')
		expect(parse("12")).toMatchObject({ dataType: DataType.Int, value: 12, text: "" })
		expect(parse("1.5")?.dataType).toBe(DataType.Float)
		expect(parse("6", float)?.dataType).toBe(DataType.Float)
		expect(parse("6", int)?.dataType).toBe(DataType.Int)
		expect(parse("6.5", int)?.dataType).toBe(DataType.Float)
		expect(parse('"x"', { ...pt("v", "0", "{}"), dataType: DataType.JSON })?.dataType).toBe(
			DataType.JSON
		)
		expect(parse("")?.dataType).toBe(DataType.Unknown)
		expect(parse("abc")).toBeUndefined()
		expect(parse('"abc')).toBeUndefined()
		expect(parse("1.")).toMatchObject({ value: 1 })
	})
})
