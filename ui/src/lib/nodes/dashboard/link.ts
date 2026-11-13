// linkUrl reads a widget's url option as the address a click opens: a full
// http or https URL, a path on this server, or a port and path on the host
// the page was loaded from, such as ":3000/d/abc" for a Grafana on the same
// machine, so the dashboard need not name the machine. Anything else, such
// as a javascript: URL, opens nothing.
export function linkUrl(
	raw: string,
	page: { protocol: string; hostname: string }
): string | undefined {
	const v = raw.trim()
	const port = /^:(\d{1,5})([/?#].*)?$/.exec(v)
	if (port) {
		const host = page.hostname.includes(":") ? `[${page.hostname}]` : page.hostname
		return `${page.protocol}//${host}:${port[1]}${port[2] ?? "/"}`
	}
	if (v.startsWith("/") && !v.startsWith("//")) {
		return v
	}
	return /^https?:\/\/[^/]/i.test(v) ? v : undefined
}
