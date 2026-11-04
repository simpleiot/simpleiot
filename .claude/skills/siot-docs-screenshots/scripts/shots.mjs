// Captures the documentation screenshots from a running siot that holds the
// demo.yaml tree. capture.sh is the usual way to run it.
//
//   node shots.mjs --out <docs dir> [--list] [name regex]
//
// Each shot is registered with `shot(name, file, fn)`: `file` is where the
// image lands under the docs directory, so the mapping from screenshot to
// document lives here and `--list` prints it for audit.py. A shot opens
// what it needs, captures, and puts the tree back the way it found it, so
// shots can run alone (`'modbus'`) or all together.
//
// Nodes are found by the tree row's accessible name, "<Type label>:
// <description>", so every description in demo.yaml has to be unique
// unless the shot narrows the search to a parent (see `within`).
import { chromium } from "../../../../ui/node_modules/@playwright/test/index.mjs"

const args = process.argv.slice(2)
const out = args.includes("--out") ? args[args.indexOf("--out") + 1] : null
const list = args.includes("--list")
const only = args.filter(
	(a, i) => !a.startsWith("--") && args[i - 1] !== "--out"
)[0]
const onlyRe = only ? new RegExp(only) : null
const base = process.env.SIOT_SHOT_URL ?? "http://localhost:15323"

const esc = (s) => s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
const item = (scope, desc) =>
	scope.getByRole("treeitem", { name: new RegExp(`^[^:]+: ${esc(desc)}$`) })
const row = (it) => it.locator(":scope > .row")

async function signIn(ctx, email, pass) {
	const page = await ctx.newPage()
	await page.goto(base + "/")
	await page.getByLabel("Email").fill(email)
	await page.getByLabel("Password").fill(pass)
	await page.getByRole("button", { name: "Sign In" }).click()
	await page.waitForURL((u) => !u.pathname.endsWith("/sign-in"))
	await page.goto(base + "/nodes")
	await page
		.getByRole("tree", { name: "Nodes" })
		.getByRole("treeitem")
		.first()
		.waitFor()
	await page.getByText("connecting...").waitFor({ state: "hidden" })
	await page.waitForTimeout(500)
	// a fresh browser session starts with the root collapsed
	await expand(page, "root")
	return page
}

async function expand(page, desc, scope = page) {
	const it = item(scope, desc)
	const b = it.getByRole("button", { name: "Expand" }).first()
	if (await b.count()) {
		await b.click()
		await page.waitForTimeout(400)
	}
	return it
}
async function collapse(page, desc, scope = page) {
	const b = item(scope, desc).getByRole("button", { name: "Collapse" }).first()
	if (await b.count()) {
		await b.click()
		await page.waitForTimeout(200)
	}
}
async function open(page, desc, scope = page) {
	const it = item(scope, desc)
	await it
		.getByRole("button", { name: /^Open details/ })
		.first()
		.click()
	await page.waitForTimeout(400)
	return it
}
async function close(page, desc, scope = page) {
	const b = item(scope, desc)
		.getByRole("button", { name: /^Close details/ })
		.first()
	if (await b.count()) {
		await b.click()
		await page.waitForTimeout(200)
	}
}
async function cancelOp(page, it) {
	const c = row(it).getByRole("button", { name: /^cancel$/i })
	if (await c.count()) await c.first().click()
}

// the shots, in the order they run; `save` is the path to write to
const shots = []
const shot = (name, file, fn) => shots.push({ name, file, fn })

// a node's row with its detail open: the editor, status, and toolbar
async function detail(page, desc, save, scope = page) {
	const it = await open(page, desc, scope)
	await row(it).screenshot({ path: save })
	await close(page, desc, scope)
}
// a slice of the tree from one node's row down to another's
async function range(page, first, last, save, lastScope = page) {
	await page.evaluate(() => window.scrollTo(0, 0))
	await page.waitForTimeout(100)
	const tree = await page.getByRole("tree", { name: "Nodes" }).boundingBox()
	const a = await row(item(page, first)).boundingBox()
	const b = await row(item(lastScope, last)).boundingBox()
	const y = a.y - 4
	await page.screenshot({
		path: save,
		fullPage: true,
		clip: { x: tree.x, y, width: tree.width, height: b.y + b.height - y + 4 },
	})
}
// a parent with a child, both open
async function parentAndChild(page, parent, child, save) {
	await expand(page, parent)
	await open(page, parent)
	const within = item(page, parent)
	await open(page, child, within)
	await range(page, parent, child, save, within)
	await close(page, child, within)
	await close(page, parent)
	await collapse(page, parent)
}

// ---- ui.md
shot("nodes", "user/images/nodes.png", async (page, save) => {
	await expand(page, "Site 1")
	await page.screenshot({
		path: save,
		clip: { x: 0, y: 0, width: 960, height: 640 },
	})
	await collapse(page, "Site 1")
})
shot("icon-arrow", "user/images/icon-arrow.png", async (page, save) => {
	await item(page, "Site 1")
		.getByRole("button", { name: "Expand" })
		.first()
		.screenshot({ path: save })
})
shot("icon-dot", "user/images/icon-dot.png", async (page, save) => {
	await item(page, "Site 1")
		.getByRole("button", { name: /^Open details/ })
		.first()
		.screenshot({ path: save })
})
for (const [name, button] of [
	["icon-delete", "Delete"],
	["icon-copy", "Copy ID"],
	["icon-paste", "Paste"],
]) {
	shot(name, `user/images/${name}.png`, async (page, save) => {
		const it = await open(page, "root")
		await row(it)
			.getByRole("button", { name: button, exact: true })
			.screenshot({ path: save })
		await close(page, "root")
	})
}
shot("node-ops", "user/images/node-ops.png", async (page, save) => {
	const it = await open(page, "root")
	await row(it)
		.getByRole("button", { name: "Raw" })
		.locator("xpath=..")
		.screenshot({ path: save })
	await close(page, "root")
})
shot("node-edit", "user/images/node-edit.png", async (page, save) => {
	await expand(page, "Site 1")
	await detail(page, "SBC", save)
	await collapse(page, "Site 1")
})
shot("node-add", "user/images/node-add.png", async (page, save) => {
	const it = await open(page, "root")
	await row(it).getByRole("button", { name: "Add", exact: true }).click()
	await page.waitForTimeout(200)
	await row(it).locator(".op").screenshot({ path: save })
	await cancelOp(page, it)
	await close(page, "root")
})
shot(
	"node-add-custom",
	"user/images/node-add-custom.png",
	async (page, save) => {
		const it = await open(page, "root")
		await row(it).getByRole("button", { name: "Add", exact: true }).click()
		await row(it).getByRole("radio", { name: "Custom" }).check()
		await row(it).locator(".op input[type=text]").fill("myCustomType")
		await row(it).locator(".op").screenshot({ path: save })
		await cancelOp(page, it)
		await close(page, "root")
	}
)
shot("paste-options", "user/images/paste-options.png", async (page, save) => {
	const src = await open(page, "Random Walk")
	await row(src).getByRole("button", { name: "Copy ID", exact: true }).click()
	await close(page, "Random Walk")
	const dst = await open(page, "Site 1")
	await row(dst).getByRole("button", { name: "Paste", exact: true }).click()
	await page.waitForTimeout(200)
	await row(dst).locator(".op").screenshot({ path: save })
	await cancelOp(page, dst)
	await close(page, "Site 1")
})
shot("node-raw", "user/images/node-raw.png", async (page, save) => {
	const it = await open(page, "admin user")
	await row(it).getByRole("button", { name: "Raw" }).click()
	await page.waitForTimeout(300)
	await row(it).screenshot({ path: save })
	await row(it).getByRole("button", { name: "Raw" }).click()
	await close(page, "admin user")
})

// ---- users-groups.md
shot("group-user", "user/images/group-user.png", async (page, save) => {
	await expand(page, "Site 1")
	await range(page, "root", "Night shift", save)
	await collapse(page, "Site 1")
})
shot("joe-nodes", "user/images/joe-nodes.png", async (_page, save, browser) => {
	const ctx = await newContext(browser)
	const page = await signIn(ctx, "joe@example.com", "joe")
	await expand(page, "Site 1")
	await page.screenshot({
		path: save,
		clip: { x: 0, y: 0, width: 960, height: 500 },
	})
	await ctx.close()
})

// ---- rules.md
shot("rules", "user/images/rules.png", async (page, save) => {
	await expand(page, "Site 1")
	await expand(page, "SBC")
	await expand(page, "Modbus client")
	await expand(page, "D5 on rule")
	await expand(page, "Device offline rule")
	await range(page, "Site 1", "Notify", save)
	await collapse(page, "Device offline rule")
	await collapse(page, "D5 on rule")
	await collapse(page, "Modbus client")
	await collapse(page, "SBC")
	await collapse(page, "Site 1")
})
// a node under Site 1 > SBC, captured with its detail open
const underSBC = (desc) => async (page, save) => {
	await expand(page, "Site 1")
	await expand(page, "SBC")
	await detail(page, desc, save)
	await collapse(page, "SBC")
	await collapse(page, "Site 1")
}
const underSite = (desc) => async (page, save) => {
	await expand(page, "Site 1")
	await detail(page, desc, save)
	await collapse(page, "Site 1")
}
shot("rule-disable", "user/images/rule-disable.png", underSite("Night shift"))
shot("rule-schedule", "user/images/rule-schedule.png", async (page, save) => {
	await expand(page, "Site 1")
	await expand(page, "Night shift")
	await detail(page, "After hours", save)
	await collapse(page, "Night shift")
	await collapse(page, "Site 1")
})
shot("rule-condition", "user/images/rule-condition.png", async (page, save) => {
	await expand(page, "Site 1")
	await expand(page, "SBC")
	await expand(page, "D5 on rule")
	await detail(page, "D5 on", save)
	await collapse(page, "D5 on rule")
	await collapse(page, "SBC")
	await collapse(page, "Site 1")
})
shot("rule-action", "user/images/rule-action.png", async (page, save) => {
	await expand(page, "Site 1")
	await expand(page, "SBC")
	await expand(page, "D5 on rule")
	await detail(page, "Turn D6 on", save)
	await collapse(page, "D5 on rule")
	await collapse(page, "SBC")
	await collapse(page, "Site 1")
})

// ---- modbus.md
shot("modbus", "user/images/modbus.png", underSBC("Modbus client"))
shot(
	"modbus-io-config",
	"user/images/modbus-io-config.png",
	async (page, save) => {
		await expand(page, "Site 1")
		await expand(page, "SBC")
		await expand(page, "Modbus client")
		await detail(page, "Holding reg @ 100", save)
		await collapse(page, "Modbus client")
		await collapse(page, "SBC")
		await collapse(page, "Site 1")
	}
)

// ---- onewire.md, shelly.md, particle.md, gpio.md, iio.md
shot("onewire-nodes", "user/images/onewire-nodes.png", async (page, save) => {
	await expand(page, "Site 1")
	await expand(page, "SBC")
	await expand(page, "rPI GPIO 4 1-Wire")
	await range(page, "rPI GPIO 4 1-Wire", "temp 2", save)
	await collapse(page, "rPI GPIO 4 1-Wire")
	await collapse(page, "SBC")
	await collapse(page, "Site 1")
})
shot("shelly", "user/images/shelly.png", async (page, save) => {
	await expand(page, "Site 1")
	await expand(page, "SBC")
	await expand(page, "Shelly")
	// children sort by description, so Office:plug is last
	await range(page, "Shelly", "Office:plug", save)
	await collapse(page, "Shelly")
	await collapse(page, "SBC")
	await collapse(page, "Site 1")
})
shot("shelly-plug", "user/images/shelly-plug.png", async (page, save) => {
	await expand(page, "Site 1")
	await expand(page, "SBC")
	await expand(page, "Shelly")
	await detail(page, "Office:plug", save)
	await collapse(page, "Shelly")
	await collapse(page, "SBC")
	await collapse(page, "Site 1")
})
shot("particle", "user/images/particle.png", underSBC("Particle"))
shot("gpio", "user/images/gpio.png", underSBC("Door switch"))
shot("iio", "user/images/iio.png", async (page, save) => {
	await expand(page, "Site 1")
	await expand(page, "SBC")
	await parentAndChild(page, "Tank level ADC", "Tank level", save)
	await collapse(page, "SBC")
	await collapse(page, "Site 1")
})

// ---- can.md, file.md, mcu.md, serial.md, browser.md, update.md
shot(
	"create-canbus-node",
	"images/create-canbus-node.png",
	async (page, save) => {
		await expand(page, "Site 1")
		const sbc = await open(page, "SBC")
		await row(sbc).getByRole("button", { name: "Add", exact: true }).click()
		await row(sbc).getByRole("radio", { name: "CAN Bus" }).check()
		await row(sbc).locator(".op").screenshot({ path: save })
		await cancelOp(page, sbc)
		await close(page, "SBC")
		await collapse(page, "Site 1")
	}
)
shot(
	"configure-canbus-node",
	"images/configure-canbus-node.png",
	async (page, save) => {
		await expand(page, "Site 1")
		await expand(page, "SBC")
		await parentAndChild(page, "CAN Test", "Test DB", save)
		await collapse(page, "SBC")
		await collapse(page, "Site 1")
	}
)
shot("file", "user/images/file.png", async (page, save) => {
	await expand(page, "Site 1")
	await expand(page, "SBC")
	await expand(page, "CAN Test")
	await detail(page, "Test DB", save)
	await collapse(page, "CAN Test")
	await collapse(page, "SBC")
	await collapse(page, "Site 1")
})
shot(
	"serial-fw-update",
	"user/images/serial-fw-update.png",
	async (page, save) => {
		await expand(page, "Site 1")
		await expand(page, "SBC")
		await parentAndChild(page, "Serial port", "FW Update", save)
		await collapse(page, "SBC")
		await collapse(page, "Site 1")
	}
)
shot("serial-port", "ref/images/serial-port.png", underSBC("Serial port"))
shot("browser", "user/images/browser.png", underSBC("Kiosk browser"))
shot("update", "user/images/update.png", underSBC("Update"))

// ---- nodes under the root: version.md, signal-generator.md, database.md,
// sync.md, messaging.md, metrics.md, gps.md, mqtt.md, configuration.md
shot("version", "ref/images/version.png", (page, save) =>
	detail(page, "root", save)
)
shot("signal-generator", "user/images/signal-generator.png", (page, save) =>
	detail(page, "Random Walk", save)
)
shot(
	"signal-generator-tags",
	"user/images/signal-generator-tags.png",
	(page, save) => detail(page, "sine wave", save)
)
shot("influx", "user/images/influx.png", (page, save) =>
	detail(page, "influx", save)
)
shot("upstream", "user/images/upstream.png", (page, save) =>
	detail(page, "upstream", save)
)
shot("twilio", "user/images/twilio.png", (page, save) =>
	detail(page, "Twilio SMS", save)
)
// the metrics table is collapsed by default; the system one is long, so
// it is cut at a height that still shows the table has started
for (const [name, desc, max] of [
	["metrics-system", "system", 1500],
	["metrics-app", "app", 0],
	["metrics-proc", "firefox proc metrics", 0],
]) {
	shot(name, `user/images/${name}.png`, async (page, save) => {
		const it = await open(page, desc)
		await row(it).locator("button.toggle").first().click()
		await page.waitForTimeout(300)
		await page.evaluate(() => window.scrollTo(0, 0))
		const b = await row(it).boundingBox()
		const height = max && b.height > max ? max : b.height
		await page.screenshot({
			path: save,
			fullPage: true,
			clip: { x: b.x, y: b.y, width: b.width, height },
		})
		await close(page, desc)
	})
}
shot("gps", "user/images/gps.png", (page, save) =>
	detail(page, "Test track", save)
)
shot("mqtt", "user/images/mqtt.png", (page, save) =>
	parentAndChild(page, "Plant data", "Tank level", save)
)
shot("provisioning", "user/images/provisioning.png", async (page, save) => {
	// the provisioning node has no description, so it is found by type
	const prov = page.getByRole("treeitem", { name: /^Provisioning:/ }).first()
	await prov.getByRole("button", { name: "Expand" }).first().click()
	await page.waitForTimeout(400)
	await prov
		.getByRole("button", { name: /^Open details/ })
		.first()
		.click()
	const file = prov.getByRole("treeitem").first()
	await file
		.getByRole("button", { name: /^Open details/ })
		.first()
		.click()
	await page.waitForTimeout(400)
	await page.evaluate(() => window.scrollTo(0, 0))
	const tree = await page.getByRole("tree", { name: "Nodes" }).boundingBox()
	const a = await row(prov).boundingBox()
	const b = await row(file).boundingBox()
	await page.screenshot({
		path: save,
		fullPage: true,
		clip: {
			x: tree.x,
			y: a.y - 4,
			width: tree.width,
			height: b.y + b.height - a.y + 8,
		},
	})
	await file
		.getByRole("button", { name: /^Close details/ })
		.first()
		.click()
	await prov
		.getByRole("button", { name: /^Close details/ })
		.first()
		.click()
	await prov.getByRole("button", { name: "Collapse" }).first().click()
})

// ---- dashboards.md: the Overview from testdata/dashboard-demo.yaml, which
// capture.sh imports as well. Last, so the trends have some history.
shot(
	"dashboard-overview",
	"user/images/dashboard-overview.png",
	async (_page, save, browser) => {
		const ctx = await browser.newContext({
			viewport: { width: 1075, height: 700 },
			deviceScaleFactor: 2,
			colorScheme: "light",
		})
		const page = await ctx.newPage()
		await page.goto(base + "/")
		await page.getByLabel("Email").fill("admin")
		await page.getByLabel("Password").fill("admin")
		await page.getByRole("button", { name: "Sign In" }).click()
		// the admin's home dashboard is the Overview, so "/" lands on it
		await page.getByRole("heading", { name: "Overview" }).waitFor()
		await page.waitForTimeout(2000)
		await page.screenshot({ path: save })
		await ctx.close()
	}
)

// ---------------------------------------------------------------------
if (list) {
	for (const s of shots) console.log(`${s.name}\t${s.file}`)
	process.exit(0)
}
if (!out) {
	console.error("usage: node shots.mjs --out <docs dir> [--list] [name regex]")
	process.exit(2)
}

// 2x for crisp text in the book; light theme so the pages match
const newContext = (browser) =>
	browser.newContext({
		viewport: { width: 960, height: 800 },
		deviceScaleFactor: 2,
		colorScheme: "light",
	})

const browser = await chromium.launch({
	channel: process.env.SIOT_SHOT_CHANNEL ?? "chrome",
})
const ctx = await newContext(browser)
const page = await signIn(ctx, "admin", "admin")
let failed = 0
for (const s of shots) {
	if (onlyRe && !onlyRe.test(s.name)) continue
	const save = `${out}/${s.file}`
	try {
		await s.fn(page, save, browser)
		console.log("ok  ", s.name, "->", s.file)
	} catch (e) {
		failed++
		console.log("FAIL", s.name, String(e).split("\n")[0])
		// start over from a known state so one failure does not sink the rest
		await page.goto(base + "/nodes")
		await page
			.getByRole("tree", { name: "Nodes" })
			.getByRole("treeitem")
			.first()
			.waitFor()
		await page.waitForTimeout(500)
		await expand(page, "root")
	}
}
await browser.close()
process.exit(failed ? 1 : 0)
