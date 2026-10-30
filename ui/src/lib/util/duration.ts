// Formats a duration in milliseconds as days, hours, minutes, and seconds,
// such as "1d 2h 3m 4s", leaving out the parts that are zero.

const sec = 1000
const min = 60 * sec
const hour = 60 * min
const day = 24 * hour

export function durationString(ms: number): string {
	const days = Math.floor(ms / day)
	const hours = Math.floor((ms % day) / hour)
	const minutes = Math.floor((ms % hour) / min)
	const seconds = Math.floor((ms % min) / sec)
	const parts: string[] = []
	if (days > 0) {
		parts.push(`${days}d`)
	}
	if (hours > 0) {
		parts.push(`${hours}h`)
	}
	if (minutes > 0) {
		parts.push(`${minutes}m`)
	}
	if (seconds > 0 || (days <= 0 && hours <= 0 && minutes <= 0)) {
		parts.push(`${seconds}s`)
	}
	return parts.join(" ")
}
