// Input sanitizers, ported from the Elm UI's UI/Sanitize.elm. Each keeps the
// valid start of what was typed and drops the rest, so a field never holds
// text that cannot become a valid value.

const isDigit = (c: string) => c >= "0" && c <= "9"

// float keeps a leading minus, digits, and one decimal point.
export function float(input: string): string {
	let out = ""
	for (const c of input) {
		if ((out.length === 0 && c === "-") || isDigit(c) || (c === "." && !out.includes("."))) {
			out += c
		} else {
			break
		}
	}
	return out
}

// time keeps the valid start of an hh:mm time. A single hour digit
// followed by ":" gets a leading zero.
export function time(input: string): string {
	let out = ""
	let i = 0
	while (i < input.length) {
		const c = input[i] as string
		switch (out.length) {
			case 0:
			case 3:
				if (!isDigit(c)) {
					return out
				}
				out += c
				break
			case 1:
				if (c === ":") {
					out = "0" + out
					continue // the colon is read again at length 2
				}
				if (!isDigit(c) || Number(out + c) > 23) {
					return out
				}
				out += c
				break
			case 2:
				if (c !== ":") {
					return out
				}
				out += c
				break
			case 4:
				if (!isDigit(c) || Number(out.slice(3) + c) > 59) {
					return out
				}
				out += c
				break
			default:
				return out
		}
		i++
	}
	return out
}

// date keeps the valid start of a YYYY-MM-DD date in this century. A single
// month digit followed by "-" gets a leading zero.
export function date(input: string): string {
	let out = ""
	let i = 0
	while (i < input.length) {
		const c = input[i] as string
		switch (out.length) {
			case 0:
				if (c !== "2") {
					return out
				}
				out += c
				break
			case 1:
				if (c !== "0") {
					return out
				}
				out += c
				break
			case 2:
			case 3:
			case 5:
			case 8:
				if (!isDigit(c)) {
					return out
				}
				out += c
				break
			case 4:
			case 7:
				if (c !== "-") {
					return out
				}
				out += c
				break
			case 6:
				if (c === "-") {
					out = out.slice(0, 5) + "0" + out.slice(5)
					continue // the dash is read again at length 7
				}
				if (!isDigit(c) || Number(out.slice(5) + c) > 12) {
					return out
				}
				out += c
				break
			case 9:
				if (!isDigit(c) || Number(out.slice(8) + c) > 31) {
					return out
				}
				out += c
				break
			default:
				return out
		}
		i++
	}
	return out
}

// parseHM accepts a complete h:mm or hh:mm time.
export function parseHM(t: string): string | null {
	const m = /^(\d{1,2}):(\d{2})$/.exec(t)
	if (!m) {
		return null
	}
	const h = Number(m[1])
	const min = Number(m[2])
	return h <= 23 && min <= 59 ? t : null
}

// parseDate accepts a complete YYYY-MM-DD date from 2023 to 2099.
export function parseDate(d: string): string | null {
	const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(d)
	if (!m) {
		return null
	}
	const y = Number(m[1])
	const mo = Number(m[2])
	const day = Number(m[3])
	return y >= 2023 && y <= 2099 && mo >= 1 && mo <= 12 && day >= 1 && day <= 31 ? d : null
}
