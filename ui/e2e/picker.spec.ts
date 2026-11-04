import { expect, test } from "@playwright/test"

// A condition picks its node by description: the search runs on the
// server, the chosen node shows as a link, and the point type list offers
// what that node carries.
test("a condition picks its node through the search", async ({ page }) => {
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

	// a variable to find, named so the search has something to match
	await root.getByRole("button", { name: "Add", exact: true }).click()
	await page.getByRole("radio", { name: "Variable" }).check()
	await page.getByRole("button", { name: "add", exact: true }).click()
	const added = tree.getByRole("treeitem", { name: "Variable: New, please edit" })
	const key = (await added.getAttribute("data-key")) ?? ""
	const variable = page.locator(`[role="treeitem"][data-key="${key}"]`)
	await variable.getByRole("button", { name: "Open details" }).click()
	const desc = variable.getByLabel("Description")
	await desc.fill("Picker target")
	await desc.press("Enter")
	await expect(variable.getByRole("button", { name: "save" })).toBeHidden()
	await variable.getByRole("button", { name: "Close details" }).click()

	// a rule with a condition
	await root.getByRole("button", { name: "Add", exact: true }).click()
	await page.getByRole("radio", { name: "Rule" }).check()
	await page.getByRole("button", { name: "add", exact: true }).click()
	const rule = tree.getByRole("treeitem", { name: "Rule: New, please edit" })
	await rule.getByRole("button", { name: "Open details" }).click()
	await rule.getByRole("button", { name: "Add", exact: true }).click()
	await page.getByRole("radio", { name: "Condition" }).check()
	await page.getByRole("button", { name: "add", exact: true }).click()
	const condition = tree.getByRole("treeitem", { name: "Condition: New, please edit" })
	await condition.getByRole("button", { name: "Open details" }).click()
	await condition.getByRole("radio", { name: "point value" }).check()

	// search, pick, and see the chip
	const node = condition.getByRole("combobox", { name: "Node" })
	await node.fill("picker")
	const option = page.getByRole("option", { name: /Picker target/ })
	await expect(option).toBeVisible()
	await node.press("ArrowDown")
	await node.press("Enter")
	const chip = condition.getByRole("link", { name: "Picker target" })
	await expect(chip).toBeVisible()
	await expect(chip).toHaveAttribute("href", /\/nodes\/[\w-]+$/)

	// the point type list offers the node's own types
	const pointType = condition.getByLabel("Point Type")
	await expect(pointType.locator("optgroup[label='on this node'] option")).toHaveCount(1)
	await pointType.selectOption("description")
	await condition.getByRole("button", { name: "save" }).click()
	await expect(condition.getByRole("button", { name: "save" })).toBeHidden()

	// the saved reference survives a reload
	await page.reload()
	const saved = page.getByRole("treeitem", { name: "Condition: New, please edit" })
	await expect(saved.getByRole("link", { name: "Picker target" })).toBeVisible()
	await expect(saved.getByLabel("Point Type")).toHaveValue("description")
})
