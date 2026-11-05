// Screenshots a dashboard from a running siot, for looking at what a node
// file built. Drives the system Chrome through Playwright from ui/.
//
//   node shot.mjs --dash "Overview" --out /path/shot.png [options]
//
//   --url http://localhost:4223   the siot HTTP port (default: SIOT_SHOT_URL
//                                 or http://localhost:4223)
//   --dash "Overview"             the dashboard's description; omit for the
//                                 dashboard list
//   --out shot.png                where the image lands (default: shot.png)
//   --user admin --pass admin     sign-in (default: admin / admin)
//   --width 1200 --height 800     viewport (default: 1200 by 800)
//   --edit                        open edit mode, so ports and frames show
//   --kiosk                       hide the header and title
//   --dark                        the dark theme
//   --wait 2000                   ms to let values arrive (default: 2000)
//
// It prints every bound element's state (data-siot-state) and the widgets
// drawn dead, so a binding that found nothing shows up in the output as
// well as in the image.
import { chromium } from "../../../../ui/node_modules/@playwright/test/index.mjs"

const args = process.argv.slice(2)
const opt = (name, fallback) => {
	const i = args.indexOf("--" + name)
	return i >= 0 && args[i + 1] !== undefined ? args[i + 1] : fallback
}
const flag = (name) => args.includes("--" + name)

const base = opt("url", process.env.SIOT_SHOT_URL ?? "http://localhost:4223")
const dash = opt("dash", "")
const out = opt("out", "shot.png")
const width = Number(opt("width", "1200"))
const height = Number(opt("height", "800"))
const wait = Number(opt("wait", "2000"))

const browser = await chromium.launch({ channel: "chrome" })
const page = await browser.newPage({
	viewport: { width, height },
	colorScheme: flag("dark") ? "dark" : "light",
})
page.on("pageerror", (e) => console.log("page error:", e.message))

await page.goto(base + "/")
// the app lands on sign-in, the home dashboard, the list, or the tree
await page.waitForURL((u) => /\/(sign-in|dash|nodes)/.test(u.pathname))
if (page.url().endsWith("/sign-in")) {
	await page.getByLabel("Email").fill(opt("user", "admin"))
	await page.getByLabel("Password").fill(opt("pass", "admin"))
	await page.getByRole("button", { name: "Sign In" }).click()
	await page.waitForURL((u) => !u.pathname.endsWith("/sign-in"))
}

await page.goto(base + "/dash")
await page.waitForTimeout(1000)
if (dash !== "") {
	const link = page.getByRole("link", { name: dash, exact: true }).first()
	if ((await link.count()) === 0) {
		const names = (await page.getByRole("link").allTextContents())
			.map((t) => t.trim())
			.filter(Boolean)
		console.log(
			`no dashboard named "${dash}"; the list shows: ${names.join(", ")}`
		)
		await browser.close()
		process.exit(1)
	}
	await link.click()
	await page.waitForURL(/\/dash\//)
	if (flag("kiosk")) {
		const u = new URL(page.url())
		u.searchParams.set("kiosk", "")
		await page.goto(u.toString())
	}
	await page.waitForTimeout(wait)
	if (flag("edit")) {
		await page.getByRole("button", { name: "edit", exact: true }).click()
		await page.waitForTimeout(500)
	}
	const dead = await page
		.locator(".dead")
		.evaluateAll((els) =>
			els.map((e) => e.textContent?.trim().split("\n")[0] ?? "").filter(Boolean)
		)
	const states = await page.evaluate(() =>
		[...document.querySelectorAll("[data-siot-bind]")].map(
			(e) =>
				`${e.getAttribute("data-siot-bind")}=${e.getAttribute("data-siot-state")}`
		)
	)
	console.log("url:", page.url())
	if (dead.length > 0) {
		console.log("widgets drawn dead:", dead.join(" | "))
	}
	if (states.length > 0) {
		console.log("drawing states:", states.join(", "))
	}
}
await page.screenshot({ path: out })
console.log("saved:", out)
await browser.close()
