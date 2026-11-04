import { expect, test } from "@playwright/test"

// A dashboard built in the tree shows its widget's live value on the
// dashboard page, and is listed under Dashboards.
test("a dashboard shows a widget bound to a variable", async ({ page }) => {
	await page.goto("/")
	await page.getByLabel("Email").fill("admin")
	await page.getByLabel("Password").fill("admin")
	await page.getByRole("button", { name: "Sign In" }).click()
	// "/" lands on a home dashboard when the user has one; the tree is at /nodes
	await expect(page).not.toHaveURL(/\/sign-in$/)
	await page.goto("/nodes")

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
	await variable.getByLabel("Description").fill("Sump level")
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
	await node.fill("sump")
	await expect(page.getByRole("option", { name: /Sump level/ }).first()).toBeVisible()
	await node.press("ArrowDown")
	await node.press("Enter")
	await expect(widget.getByRole("link", { name: "Sump level" })).toBeVisible()
	await widget.getByLabel("Width").fill("4")
	await widget.getByRole("button", { name: "save" }).click()
	await expect(widget.getByRole("button", { name: "save" })).toBeHidden()

	// the dashboard page shows the value, and the list names the dashboard
	await page.goto(href)
	await expect(page.getByRole("heading", { name: "Lift station" })).toBeVisible()
	await expect(page.getByText("42.5")).toBeVisible()
	await expect(page.getByText("Level", { exact: true })).toBeVisible()

	// the tile is a control: a click opens an edit box, and Enter writes
	await page.getByRole("button", { name: "Edit Level" }).click()
	const box = page.getByRole("spinbutton", { name: "Set Level" })
	await box.fill("50")
	await box.press("Enter")
	await expect(page.getByText("50.0")).toBeVisible()

	// edit mode adds a widget and opens its settings
	await page.getByRole("button", { name: "edit", exact: true }).click()
	await page.getByRole("button", { name: "add widget" }).click()
	const panel = page.getByRole("complementary", { name: "Widget settings" })
	await expect(panel.getByLabel("Label")).toBeVisible()
	await panel.getByLabel("Label").fill("Second")
	await panel.getByRole("button", { name: "save" }).click()
	await expect(panel.getByRole("button", { name: "save" })).toBeHidden()
	await expect(page.locator("main").getByText("Second", { exact: true }).first()).toBeVisible()

	// the address names the selected widget, and opens its panel again
	await expect(page).toHaveURL(/\/dash\/[\w-]+\/[\w-]+$/)
	const widgetUrl = page.url()

	// duplicate adds a copy and opens it; copy and paste add another
	await panel.getByRole("button", { name: "duplicate" }).click()
	await expect(page).not.toHaveURL(widgetUrl)
	await expect(panel.getByLabel("Label")).toHaveValue("Second")
	await panel.getByRole("button", { name: "copy" }).click()
	await page.getByRole("button", { name: /^paste / }).click()
	await expect(page.locator("main").getByText("Second", { exact: true })).toHaveCount(4)
	await page.goto(widgetUrl)
	await page.reload()
	await expect(
		page.getByRole("complementary", { name: "Widget settings" }).getByLabel("Label")
	).toHaveValue("Second")
	await page.getByRole("button", { name: "done" }).click()
	await expect(page).toHaveURL(/\/dash\/[\w-]+$/)

	// going to the tree and back lands on the widget link with its panel
	await page.goto(widgetUrl)
	await page.getByRole("link", { name: "in the tree" }).click()
	await expect(page.getByRole("tree", { name: "Nodes" })).toBeVisible()
	await page.goBack()
	await expect(page).toHaveURL(widgetUrl)
	await expect(
		page.getByRole("complementary", { name: "Widget settings" }).getByLabel("Label")
	).toHaveValue("Second")
	await page.getByRole("button", { name: "done" }).click()

	// signed out, the link asks for sign-in and then lands where it pointed
	await page.getByRole("button", { name: /sign out/ }).click()
	await page.goto(widgetUrl)
	await expect(page).toHaveURL(/\/sign-in\?next=/)
	await page.getByLabel("Email").fill("admin")
	await page.getByLabel("Password").fill("admin")
	await page.getByRole("button", { name: "Sign In" }).click()
	await expect(page).toHaveURL(widgetUrl)
	await expect(
		page.getByRole("complementary", { name: "Widget settings" }).getByLabel("Label")
	).toHaveValue("Second")
	await page.getByRole("button", { name: "done" }).click()

	await page.getByRole("link", { name: "Dashboards" }).click()
	await expect(page.locator(`main a[href="${href}"]`)).toHaveText("Lift station")

	// the home dashboard, set on the user, is where "/" lands
	await page.getByRole("link", { name: "Nodes" }).click()
	const user = tree.getByRole("treeitem", { name: /^User:/ }).first()
	await user.getByRole("button", { name: "Open details" }).click()
	// a home dashboard set earlier, such as by the demo file, is cleared first
	if ((await user.getByRole("button", { name: "Clear" }).count()) > 0) {
		await user.getByRole("button", { name: "Clear" }).click()
	}
	const home = user.getByRole("combobox", { name: "Home dashboard" })
	await home.fill("lift")
	await expect(page.getByRole("option", { name: /Lift station/ }).first()).toBeVisible()
	await home.press("ArrowDown")
	await home.press("Enter")
	await user.getByRole("button", { name: "save" }).click()
	await expect(user.getByRole("button", { name: "save" })).toBeHidden()
	await page.goto("/")
	await expect(page).toHaveURL(/\/dash\/[\w-]+$/)
	await expect(page.getByRole("heading", { name: "Lift station" })).toBeVisible()

	// clear it again, so other tests land where they expect
	await page.goto("/nodes")
	const user2 = tree.getByRole("treeitem", { name: /^User:/ }).first()
	await expect(user2).toBeVisible()
	// the open detail is remembered across the reload
	if ((await user2.getByRole("button", { name: "Clear" }).count()) === 0) {
		await user2.getByRole("button", { name: "Open details" }).click()
	}
	await user2.getByRole("button", { name: "Clear" }).click()
	await user2.getByRole("button", { name: "save" }).click()
	await expect(user2.getByRole("button", { name: "save" })).toBeHidden()
})
