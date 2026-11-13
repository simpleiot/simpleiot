import { expect, test, type Page } from "@playwright/test"

async function signIn(page: Page) {
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
	return { tree, root }
}

// add puts a new node of a type under a parent row and opens it
async function add(page: Page, parent: ReturnType<Page["locator"]>, type: string) {
	const tree = page.getByRole("tree", { name: "Nodes" })
	await parent.getByRole("button", { name: "Add", exact: true }).click()
	await page.getByRole("radio", { name: type }).check()
	await page.getByRole("button", { name: "add", exact: true }).click()
	const row = tree.getByRole("treeitem", { name: `${type}: New, please edit` })
	const key = (await row.getAttribute("data-key")) ?? ""
	const node = page.locator(`[role="treeitem"][data-key="${key}"]`)
	await node.getByRole("button", { name: "Open details" }).click()
	return { node, id: key.split("/").pop() ?? "" }
}

async function save(node: ReturnType<Page["locator"]>) {
	await node.getByRole("button", { name: "save" }).click()
	await expect(node.getByRole("button", { name: "save" })).toBeHidden()
}

// navigateTo binds a widget's navigate binding to a dashboard by name
async function navigateTo(page: Page, widget: ReturnType<Page["locator"]>, name: string) {
	const picker = widget.getByRole("combobox", { name: "navigate node" })
	await picker.fill(name)
	await expect(page.getByRole("option", { name: new RegExp(name) }).first()).toBeVisible()
	await picker.press("ArrowDown")
	await picker.press("Enter")
	await expect(widget.getByRole("link", { name })).toBeVisible()
}

test("a widget with a navigate binding opens its dashboard", async ({ page }) => {
	const { root } = await signIn(page)

	const { node: target, id: targetId } = await add(page, root, "Dashboard")
	await target.getByLabel("Description").fill("Link target")
	await save(target)
	await target.getByRole("button", { name: "Close details" }).click()

	const { node: origin } = await add(page, root, "Dashboard")
	await origin.getByLabel("Description").fill("Link origin")
	await origin.getByRole("radio", { name: "canvas" }).check()
	await save(origin)
	const href = (await origin.getByRole("link", { name: "open" }).getAttribute("href")) ?? ""

	// a value tile, which is not a control here, and a button
	const { node: tile } = await add(page, origin, "Widget")
	await tile.getByLabel("Label").fill("Go reading")
	await tile.getByLabel("Component").selectOption("value")
	await tile.getByLabel("X:", { exact: true }).fill("20")
	await tile.getByLabel("Y:", { exact: true }).fill("20")
	await navigateTo(page, tile, "Link target")
	await save(tile)
	await tile.getByRole("button", { name: "Close details" }).click()

	const { node: button } = await add(page, origin, "Widget")
	await button.getByLabel("Label").fill("To target")
	await button.getByLabel("Component").selectOption("button")
	await button.getByLabel("X:", { exact: true }).fill("300")
	await button.getByLabel("Y:", { exact: true }).fill("20")
	await navigateTo(page, button, "Link target")
	await save(button)

	await page.setViewportSize({ width: 1200, height: 800 })
	const toTarget = new RegExp(`/dash/${targetId}`)

	// in edit mode a click selects the widget and goes nowhere
	await page.goto(href)
	await expect(page.getByRole("heading", { name: "Link origin" })).toBeVisible()
	await page.getByRole("button", { name: "edit" }).click()
	await page.locator(".equipment", { hasText: "Go reading" }).click()
	await expect(page.locator(".equipment.selected", { hasText: "Go reading" })).toHaveCount(1)
	await expect(page).not.toHaveURL(toTarget)

	// in view mode a click on the tile opens the target
	await page.goto(href)
	await page.getByRole("link", { name: /Go reading/ }).click()
	await expect(page).toHaveURL(toTarget)
	await expect(page.getByRole("heading", { name: "Link target" })).toBeVisible()

	// the button opens it from the keyboard too
	await page.goto(href)
	const link = page.getByRole("link", { name: "To target" })
	await link.focus()
	await page.keyboard.press("Enter")
	await expect(page).toHaveURL(toTarget)
})
