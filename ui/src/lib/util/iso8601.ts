// Formats a time as an ISO 8601 date and time with no zone suffix, in the
// browser's zone or UTC.

const two = (n: number) => String(n).padStart(2, "0")

export function toDateTimeString(t: Date | string | number, utc = false): string {
	const d = new Date(t)
	if (utc) {
		return `${d.getUTCFullYear()}-${two(d.getUTCMonth() + 1)}-${two(d.getUTCDate())}T${two(
			d.getUTCHours()
		)}:${two(d.getUTCMinutes())}:${two(d.getUTCSeconds())}`
	}
	return `${d.getFullYear()}-${two(d.getMonth() + 1)}-${two(d.getDate())}T${two(
		d.getHours()
	)}:${two(d.getMinutes())}:${two(d.getSeconds())}`
}
