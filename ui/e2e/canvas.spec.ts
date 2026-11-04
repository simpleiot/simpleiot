import { expect, test } from "@playwright/test"

// A canvas dashboard draws equipment and a link between their ports.
test("a canvas dashboard draws a pump, a tank, and a link", async ({ page }) => {
	await page.goto("/")
	await page.getByLabel("Email").fill("admin")
	await page.getByLabel("Password").fill("admin")
	await page.getByRole("button", { name: "Sign In" }).click()
	await expect(page).not.toHaveURL(/\/sign-in$/)
	await page.goto("/nodes")

	const tree = page.getByRole("tree", { name: "Nodes" })
	const root = tree.getByRole("treeitem").first()
	await expect(root).toBeVisible()
	await expect(page.getByText("connecting...")).toBeHidden()
	if ((await root.getByRole("button", { name: "Collapse" }).count()) === 0) {
		await root.getByRole("button", { name: "Expand" }).first().click()
	}
	if ((await root.getByRole("button", { name: "Add", exact: true }).count()) === 0) {
		await root.getByRole("button", { name: "Open details" }).first().click()
	}

	// a canvas dashboard
	await root.getByRole("button", { name: "Add", exact: true }).click()
	await page.getByRole("radio", { name: "Dashboard" }).check()
	await page.getByRole("button", { name: "add", exact: true }).click()
	const drow = tree.getByRole("treeitem", { name: "Dashboard: New, please edit" })
	const dkey = (await drow.getAttribute("data-key")) ?? ""
	const dashboard = page.locator(`[role="treeitem"][data-key="${dkey}"]`)
	await dashboard.getByRole("button", { name: "Open details" }).click()
	await dashboard.getByLabel("Description").fill("Pump station")
	await dashboard.getByRole("radio", { name: "canvas" }).check()
	await dashboard.getByRole("button", { name: "save" }).click()
	await expect(dashboard.getByRole("button", { name: "save" })).toBeHidden()
	const href = (await dashboard.getByRole("link", { name: "open" }).getAttribute("href")) ?? ""

	// two pieces of equipment, by component, and a link between their ports
	const ids: string[] = []
	for (const [component, x] of [
		["pump", "40"],
		["tank", "320"],
	] as const) {
		await dashboard.getByRole("button", { name: "Add", exact: true }).click()
		await page.getByRole("radio", { name: "Widget" }).check()
		await page.getByRole("button", { name: "add", exact: true }).click()
		const wrow = tree.getByRole("treeitem", { name: "Widget: New, please edit" })
		const wkey = (await wrow.getAttribute("data-key")) ?? ""
		ids.push(wkey.split("/").pop() ?? "")
		const widget = page.locator(`[role="treeitem"][data-key="${wkey}"]`)
		await widget.getByRole("button", { name: "Open details" }).click()
		await widget.getByLabel("Label").fill(component)
		await widget.getByLabel("Component").selectOption(component)
		await widget.getByLabel("X:", { exact: true }).fill(x)
		await widget.getByLabel("Y:", { exact: true }).fill("120")
		await widget.getByRole("button", { name: "save" }).click()
		await expect(widget.getByRole("button", { name: "save" })).toBeHidden()
		await widget.getByRole("button", { name: "Close details" }).click()
	}
	await dashboard.getByRole("button", { name: "Add", exact: true }).click()
	await page.getByRole("radio", { name: "Link" }).check()
	await page.getByRole("button", { name: "add", exact: true }).click()
	const lrow = tree.getByRole("treeitem", { name: "Link: New, please edit" })
	const lkey = (await lrow.getAttribute("data-key")) ?? ""
	const link = page.locator(`[role="treeitem"][data-key="${lkey}"]`)
	await link.getByRole("button", { name: "Open details" }).click()
	await link.getByLabel("From").fill(`${ids[0]}.discharge`)
	await link.getByLabel("To").fill(`${ids[1]}.in`)
	await link.getByRole("radio", { name: "orthogonal" }).check()
	await link.getByRole("button", { name: "save" }).click()
	await expect(link.getByRole("button", { name: "save" })).toBeHidden()

	// the dashboard page draws both and the pipe between them
	await page.setViewportSize({ width: 1200, height: 800 })
	await page.goto(href)
	await expect(page.getByRole("heading", { name: "Pump station" })).toBeVisible()
	await expect(page.getByRole("img", { name: /^Pump pump/ })).toBeVisible()
	await expect(page.getByRole("img", { name: /^Tank tank/ })).toBeVisible()
	const wall = page.locator("svg.links path.wall")
	await expect(wall).toHaveCount(1)
	const d = (await wall.getAttribute("d")) ?? ""
	expect(d.split("L").length).toBeGreaterThan(2)
})
