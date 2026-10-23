// Converts the times of day and dates of a schedule between UTC, as the
// points store them, and the browser's zone, as the form shows them. Ported
// from the Elm UI's Utils/Time.elm. offset is the zone's offset from UTC in
// minutes.

export interface Schedule {
	startTime: string
	endTime: string
	// weekdays are Go's time.Weekday numbers: 0 is Sunday
	weekdays: number[]
	dates: string[]
	// dateCount is how many date points the node has, so dates removed in
	// the form can be deleted
	dateCount: number
}

const minutesInDay = 24 * 60

function parseMinutes(t: string): number {
	const m = /^(\d+):(\d+)$/.exec(t.trim())
	return m ? Number(m[1]) * 60 + Number(m[2]) : 0
}

function formatMinutes(m: number): string {
	const two = (n: number) => String(n).padStart(2, "0")
	return `${two(Math.floor(m / 60))}:${two(m % 60)}`
}

function normalize(m: number): number {
	if (m < 0) {
		return m + minutesInDay
	}
	if (m >= minutesInDay) {
		return m - minutesInDay
	}
	return m
}

export function toLocal(offset: number, t: string): string {
	return formatMinutes(normalize(parseMinutes(t) + offset))
}

export function toUTC(offset: number, t: string): string {
	return formatMinutes(normalize(parseMinutes(t) - offset))
}

// shift moves a time by offset and reports how many days it crossed.
function shift(offset: number, t: string): [string, number] {
	const from = parseMinutes(t)
	const to = normalize(from + offset)
	let days = 0
	if (offset > 0) {
		days = to < from ? 1 : 0
	} else if (to > from) {
		days = -1
	}
	return [formatMinutes(to), days]
}

function shiftWeekday(days: number, wd: number): number {
	const n = wd + days
	if (n < 0) {
		return 6
	}
	if (n > 6) {
		return 0
	}
	return n
}

function shiftDate(days: number, d: string): string {
	const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(d)
	if (!m) {
		return d
	}
	const t = new Date(Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3]) + days))
	if (Number.isNaN(t.valueOf())) {
		return d
	}
	return t.toISOString().slice(0, 10)
}

const numeric = (a: number, b: number) => a - b

export function scheduleToLocal(offset: number, s: Schedule): Schedule {
	const [startTime, days] = shift(offset, s.startTime)
	return {
		startTime,
		endTime: toLocal(offset, s.endTime),
		weekdays: s.weekdays.map((w) => shiftWeekday(days, w)).sort(numeric),
		dates: s.dates.map((d) => shiftDate(days, d)),
		dateCount: s.dateCount,
	}
}

export function scheduleToUTC(offset: number, s: Schedule): Schedule {
	const [startTime, days] = shift(-offset, s.startTime)
	return {
		startTime,
		endTime: toUTC(offset, s.endTime),
		weekdays: s.weekdays.map((w) => shiftWeekday(days, w)).sort(numeric),
		dates: s.dates
			.slice()
			.sort()
			.map((d) => shiftDate(days, d)),
		dateCount: s.dateCount,
	}
}
