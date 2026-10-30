import { describe, expect, test } from "vitest"
import { durationString } from "./duration.ts"
import * as sanitize from "./sanitize.ts"
import { scheduleToLocal, scheduleToUTC, toLocal, toUTC, type Schedule } from "./time.ts"

describe("durationString", () => {
	test("days, hours, minutes, seconds", () => {
		expect(durationString(1234123232)).toBe("14d 6h 48m 43s")
	})
	test("seconds only", () => {
		expect(durationString(45 * 1000)).toBe("45s")
	})
	test("1s over a day", () => {
		expect(durationString(1000 * 60 * 60 * 24 + 1000)).toBe("1d 1s")
	})
})

describe("sanitize.date", () => {
	test.each([
		["2023-06-01", "2023-06-01"],
		["2023-066", "2023-06"],
		["2023-6-01", "2023-06-01"],
		["3000", ""],
		["2300", "2"],
		["2023-61", "2023-6"],
		["2023-12-31", "2023-12-31"],
		["2023-13-31", "2023-1"],
		["2099-12-32", "2099-12-3"],
	])("%s", (input, want) => {
		expect(sanitize.date(input)).toBe(want)
	})
})

describe("sanitize.parseDate", () => {
	test("valid date", () => {
		expect(sanitize.parseDate("2023-01-23")).toBe("2023-01-23")
	})
	test.each(["202-01-23", "2023-1-23", "2023-11-8"])("%s is invalid", (d) => {
		expect(sanitize.parseDate(d)).toBeNull()
	})
})

describe("sanitize.parseHM", () => {
	test.each([
		["1:2", null],
		["12:43", "12:43"],
		["1:02", "1:02"],
		["01:02", "01:02"],
		["01:60", null],
		["23:15", "23:15"],
		["24:23", null],
		["0:00", "0:00"],
		["00:00", "00:00"],
	])("%s", (t, want) => {
		expect(sanitize.parseHM(t)).toBe(want)
	})
})

describe("sanitize.time", () => {
	test.each([
		["12:34", "12:34"],
		["1:", "01:"],
		["24", "2"],
		["12:61", "12:6"],
		["9:05x", "09:05"],
	])("%s", (t, want) => {
		expect(sanitize.time(t)).toBe(want)
	})
})

describe("sanitize.float", () => {
	test.each([
		["1.2", "1.2"],
		[".2", ".2"],
		["1.2a", "1.2"],
		["-1.2", "-1.2"],
		["1-2", "1"],
	])("%s", (input, want) => {
		expect(sanitize.float(input)).toBe(want)
	})
})

describe("toLocal and toUTC", () => {
	test.each([
		[-240, "5:00", "01:00"],
		[-240, "1:00", "21:00"],
		[240, "5:00", "09:00"],
		[240, "1:00", "05:00"],
	])("toLocal %i %s", (offset, t, want) => {
		expect(toLocal(offset, t)).toBe(want)
	})
	test.each([
		[-240, "1:00", "05:00"],
		[-240, "22:00", "02:00"],
		[240, "1:00", "21:00"],
		[240, "22:00", "18:00"],
	])("toUTC %i %s", (offset, t, want) => {
		expect(toUTC(offset, t)).toBe(want)
	})
})

const sched = (
	startTime: string,
	endTime: string,
	weekdays: number[],
	dates: string[] = []
): Schedule => ({ startTime, endTime, weekdays, dates, dateCount: dates.length })

describe("schedules", () => {
	test("toLocal no weekday change", () => {
		expect(scheduleToLocal(-240, sched("05:00", "08:00", [2, 3]))).toEqual(
			sched("01:00", "04:00", [2, 3])
		)
	})
	test("toLocal with weekday change", () => {
		expect(scheduleToLocal(-240, sched("02:00", "08:00", [0, 2, 3]))).toEqual(
			sched("22:00", "04:00", [1, 2, 6])
		)
	})
	test("toLocal with weekday change, positive offset", () => {
		expect(scheduleToLocal(240, sched("22:00", "02:00", [2, 3, 6]))).toEqual(
			sched("02:00", "06:00", [0, 3, 4])
		)
	})
	test("toLocal with no date change", () => {
		expect(scheduleToLocal(-240, sched("10:00", "12:00", [], ["2023-01-01"]))).toEqual(
			sched("06:00", "08:00", [], ["2023-01-01"])
		)
	})
	test("toLocal with date change", () => {
		expect(scheduleToLocal(-240, sched("2:00", "12:00", [], ["2023-01-01"]))).toEqual(
			sched("22:00", "08:00", [], ["2022-12-31"])
		)
	})
	test("toUTC with weekday change", () => {
		expect(scheduleToUTC(-240, sched("22:00", "02:00", [1, 2, 6]))).toEqual(
			sched("02:00", "06:00", [0, 2, 3])
		)
	})
	test("toUTC with weekday change, positive offset", () => {
		expect(scheduleToUTC(240, sched("02:00", "06:00", [0, 3, 4]))).toEqual(
			sched("22:00", "02:00", [2, 3, 6])
		)
	})
	test("toUTC with no date change", () => {
		expect(scheduleToUTC(-240, sched("02:00", "06:00", [], ["2022-12-31"]))).toEqual(
			sched("06:00", "10:00", [], ["2022-12-31"])
		)
	})
	test("toUTC with date change", () => {
		expect(scheduleToUTC(-240, sched("22:00", "23:00", [], ["2022-12-31"]))).toEqual(
			sched("02:00", "03:00", [], ["2023-01-01"])
		)
	})
})
