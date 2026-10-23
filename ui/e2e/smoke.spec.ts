import { expect, test } from "@playwright/test"

// Signs in with the default account of a fresh instance, adds a node,
// edits its description, and checks the edit is saved.
test("sign in, add a node, edit it, and see it saved", async ({ page }) => {
	await page.goto("/")
	await expect(page).toHaveURL(/\/sign-in$/)
	await page.getByLabel("Email").fill("admin")
	await page.getByLabel("Password").fill("admin")
	await page.getByRole("button", { name: "Sign In" }).click()

	const tree = page.getByRole("tree", { name: "Nodes" })
	const root = tree.getByRole("treeitem").first()
	await expect(root).toBeVisible()
	await expect(page.getByText("connecting...")).toBeHidden()

	// open the root and add a variable under it
	await root.getByRole("button", { name: "Expand" }).first().click()
	await root.getByRole("button", { name: "Open details" }).first().click()
	await root.getByRole("button", { name: "Add", exact: true }).click()
	await page.getByRole("radio", { name: "Variable" }).check()
	await page.getByRole("button", { name: "add", exact: true }).click()

	const added = tree.getByRole("treeitem", { name: "Variable: New, please edit" })
	await expect(added).toBeVisible()
	// the row's name follows its description, so hold it by its key
	const key = (await added.getAttribute("data-key")) ?? ""
	const variable = page.locator(`[role="treeitem"][data-key="${key}"]`)
	await variable.getByRole("button", { name: "Open details" }).click()
	await expect(page).toHaveURL(/\/nodes\/[\w-]+$/)

	const desc = variable.getByLabel("Description")
	await desc.fill("Tank level")
	await desc.press("Enter")
	await expect(variable.getByRole("button", { name: "save" })).toBeHidden()

	// a reload of the node's link shows the saved description, opened
	await page.reload()
	const saved = page.getByRole("treeitem", { name: "Variable: Tank level" })
	await expect(saved).toBeVisible()
	await expect(saved.getByLabel("Description")).toHaveValue("Tank level")
})
