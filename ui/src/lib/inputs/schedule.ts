// The schedule a condition node holds as points: start and end times,
// weekdays, and dates, all in UTC. Ported from the Elm UI's
// UI/NodeInputs.elm.

import { DataType, getText, getValue, newTombstone, sorted, type Point } from "#lib/siot/point.ts"
import { PointType } from "#lib/siot/schema.ts"
import { parseDate, parseHM } from "#lib/util/sanitize.ts"
import { scheduleToLocal, scheduleToUTC, type Schedule } from "#lib/util/time.ts"

export const weekdays = [0, 1, 2, 3, 4, 5, 6] as const

export function pointsToSchedule(points: readonly Point[]): Schedule {
	const datePoints = sorted(points.filter((p) => p.type === PointType.Date && p.tombstone === 0))
	return {
		startTime: getText(points, PointType.Start),
		endTime: getText(points, PointType.End),
		weekdays: weekdays.filter((d) => getValue(points, PointType.Weekday, String(d)) === 1),
		dates: datePoints.map((p) => p.text),
		dateCount: datePoints.length,
	}
}

export function scheduleToPoints(time: string, s: Schedule): Point[] {
	const text = (type: string, key: string, t: string): Point => ({
		type,
		key,
		time,
		dataType: DataType.String,
		value: 0,
		text: t,
		tombstone: 0,
	})
	const pts: Point[] = [
		text(PointType.Start, "0", s.startTime),
		text(PointType.End, "0", s.endTime),
	]
	for (const d of weekdays) {
		pts.push({
			type: PointType.Weekday,
			key: String(d),
			time,
			dataType: DataType.Float,
			value: s.weekdays.includes(d) ? 1 : 0,
			text: "",
			tombstone: 0,
		})
	}
	s.dates.forEach((d, i) => pts.push(text(PointType.Date, String(i), d)))
	// dates removed in the form are deleted, so the list stays dense
	for (let i = s.dates.length; i < s.dateCount; i++) {
		pts.push(newTombstone(PointType.Date, String(i), time))
	}
	return pts
}

const valid = (s: Schedule) =>
	parseHM(s.startTime) !== null &&
	parseHM(s.endTime) !== null &&
	s.dates.every((d) => d === "" || parseDate(d) !== null)

// toLocal and toUTC convert only a schedule whose times and dates are all
// valid; one being typed is left as it is.
export const toLocal = (offset: number, s: Schedule) => (valid(s) ? scheduleToLocal(offset, s) : s)
export const toUTC = (offset: number, s: Schedule) => (valid(s) ? scheduleToUTC(offset, s) : s)
