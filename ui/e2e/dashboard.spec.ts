import { expect, test } from "@playwright/test"

// A dashboard built in the tree shows its widget's live value on the
// dashboard page, and is listed under Dashboards.
test("a dashboard shows a widget bound to a variable", async ({ page }) => {
	await page.goto("/")
	await page.getByLabel("Email").fill("admin")
	await page.getByLabel("Password").fill("admin")
	await page.getByRole("button", { name: "Sign In" }).click()

	const tree = page.getByRole("tree", { name: "Nodes" })
	const root = tree.getByRole("treeitem").first()
	await expect(root).toBeVisible()
	await expect(page.getByText("connecting...")).toBeHidden()
	await root.getByRole("button", { name: "Expand" }).first().click()
	await root.getByRole("button", { name: "Open details" }).first().click()

	// a variable with a value to show
	await root.getByRole("button", { name: "Add", exact: true }).click()
	await page.getByRole("radio", { name: "Variable" }).check()
	await page.getByRole("button", { name: "add", exact: true }).click()
	const added = tree.getByRole("treeitem", { name: "Variable: New, please edit" })
	const vkey = (await added.getAttribute("data-key")) ?? ""
	const variable = page.locator(`[role="treeitem"][data-key="${vkey}"]`)
	await variable.getByRole("button", { name: "Open details" }).click()
	await variable.getByLabel("Description").fill("Tank level")
	await variable.getByRole("radio", { name: "number" }).check()
	await variable.getByLabel("Value").fill("42.5")
	await variable.getByRole("button", { name: "save" }).click()
	await expect(variable.getByRole("button", { name: "save" })).toBeHidden()
	await variable.getByRole("button", { name: "Close details" }).click()

	// a dashboard with one widget bound to it
	await root.getByRole("button", { name: "Add", exact: true }).click()
	await page.getByRole("radio", { name: "Dashboard" }).check()
	await page.getByRole("button", { name: "add", exact: true }).click()
	const dash = tree.getByRole("treeitem", { name: "Dashboard: New, please edit" })
	const dkey = (await dash.getAttribute("data-key")) ?? ""
	const dashboard = page.locator(`[role="treeitem"][data-key="${dkey}"]`)
	await dashboard.getByRole("button", { name: "Open details" }).click()
	await dashboard.getByLabel("Description").fill("Lift station")
	await dashboard.getByRole("button", { name: "save" }).click()
	await expect(dashboard.getByRole("button", { name: "save" })).toBeHidden()
	const open = dashboard.getByRole("link", { name: "open" })
	const href = (await open.getAttribute("href")) ?? ""
	expect(href).toMatch(/\/dash\/[\w-]+$/)

	await dashboard.getByRole("button", { name: "Add", exact: true }).click()
	await page.getByRole("radio", { name: "Widget" }).check()
	await page.getByRole("button", { name: "add", exact: true }).click()
	const wrow = tree.getByRole("treeitem", { name: "Widget: New, please edit" })
	const wkey = (await wrow.getAttribute("data-key")) ?? ""
	const widget = page.locator(`[role="treeitem"][data-key="${wkey}"]`)
	await widget.getByRole("button", { name: "Open details" }).click()
	await widget.getByLabel("Label").fill("Level")
	const node = widget.getByRole("combobox", { name: "value node" })
	await node.fill("tank")
	await expect(page.getByRole("option", { name: /Tank level/ }).first()).toBeVisible()
	await node.press("ArrowDown")
	await node.press("Enter")
	await expect(widget.getByRole("link", { name: "Tank level" })).toBeVisible()
	await widget.getByLabel("Width").fill("4")
	await widget.getByRole("button", { name: "save" }).click()
	await expect(widget.getByRole("button", { name: "save" })).toBeHidden()

	// the dashboard page shows the value, and the list names the dashboard
	await page.goto(href)
	await expect(page.getByRole("heading", { name: "Lift station" })).toBeVisible()
	await expect(page.getByText("42.5")).toBeVisible()
	await expect(page.getByText("Level", { exact: true })).toBeVisible()

	await page.getByRole("link", { name: "Dashboards" }).click()
	await expect(page.locator(`main a[href="${href}"]`)).toHaveText("Lift station")
})
