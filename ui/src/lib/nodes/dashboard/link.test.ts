import { describe, expect, test } from "vitest"
import { linkUrl } from "./link.ts"

describe("linkUrl", () => {
	const page = { protocol: "http:", hostname: "192.168.1.20" }

	test("a port opens that port on the page's host", () => {
		expect(linkUrl(":3000/d/abc/pumps?orgId=1", page)).toBe(
			"http://192.168.1.20:3000/d/abc/pumps?orgId=1"
		)
		expect(linkUrl(" :3000 ", page)).toBe("http://192.168.1.20:3000/")
		expect(linkUrl(":3000", { protocol: "https:", hostname: "::1" })).toBe("https://[::1]:3000/")
	})

	test("full URLs and paths open as given", () => {
		expect(linkUrl("https://grafana.example.com/d/abc", page)).toBe(
			"https://grafana.example.com/d/abc"
		)
		expect(linkUrl("/grafana/d/abc", page)).toBe("/grafana/d/abc")
	})

	test("anything else opens nothing", () => {
		for (const v of ["", "javascript:alert(1)", "//evil.example", "grafana", ":abc", "data:x"]) {
			expect(linkUrl(v, page)).toBeUndefined()
		}
	})
})
