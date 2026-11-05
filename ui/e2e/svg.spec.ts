import { expect, test } from "@playwright/test"

// A lamp drawing uploaded to a file node, bound on a dashboard: the
// widget editor lists the binding the drawing declares, the lamp follows
// the variable, and a click on the glass flips it.
const lamp = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 130">
  <style>
    .glass { fill: #ddd; }
    .glass[data-siot-state="on"] { fill: #ffd54a; }
  </style>
  <circle class="glass" cx="50" cy="48" r="34" data-siot-bind="lit" data-siot-toggle />
  <rect x="36" y="84" width="28" height="22" fill="#777" />
</svg>`

test("a drawing on a dashboard follows and flips a variable", async ({ page }) => {
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

	// the drawing, uploaded to a file node through the row's hidden input
	await root.getByRole("button", { name: "Add", exact: true }).click()
	await page.getByRole("radio", { name: "File" }).check()
	await page.getByRole("button", { name: "add", exact: true }).click()
	const frow = tree.getByRole("treeitem", { name: "File: New, please edit" })
	const fkey = (await frow.getAttribute("data-key")) ?? ""
	const file = page.locator(`[role="treeitem"][data-key="${fkey}"]`)
	await file.getByRole("button", { name: "Open details" }).click()
	await file.getByLabel("Description").fill("Lamp drawing")
	await file.getByRole("button", { name: "save" }).click()
	await expect(file.getByRole("button", { name: "save" })).toBeHidden()
	await file.locator('input[type="file"]').setInputFiles({
		name: "lamp.svg",
		mimeType: "image/svg+xml",
		buffer: Buffer.from(lamp),
	})
	await expect(file.getByText("File name: lamp.svg")).toBeVisible()
	// the contents open in the code editor, highlighted as XML
	await file.getByRole("button", { name: "Edit contents" }).click()
	const code = file.getByRole("textbox", { name: "Contents" })
	await expect(code).toContainText('data-siot-bind="lit"')
	await expect(code.locator(".cm-line span[class]").first()).toBeVisible()
	await file.getByRole("button", { name: "Close contents" }).click()
	await file.getByRole("button", { name: "Close details" }).click()

	// the switch the lamp follows
	await root.getByRole("button", { name: "Add", exact: true }).click()
	await page.getByRole("radio", { name: "Variable" }).check()
	await page.getByRole("button", { name: "add", exact: true }).click()
	const vrow = tree.getByRole("treeitem", { name: "Variable: New, please edit" })
	const vkey = (await vrow.getAttribute("data-key")) ?? ""
	const variable = page.locator(`[role="treeitem"][data-key="${vkey}"]`)
	await variable.getByRole("button", { name: "Open details" }).click()
	await variable.getByLabel("Description").fill("Lamp switch")
	await variable.getByRole("radio", { name: "On/Off" }).check()
	await variable.getByRole("button", { name: "save" }).click()
	await expect(variable.getByRole("button", { name: "save" })).toBeHidden()
	// the lamp starts lit
	const sw = variable.getByRole("switch", { name: "Value" })
	await sw.click()
	await expect(sw).toHaveAttribute("aria-checked", "true")
	await variable.getByRole("button", { name: "save" }).click()
	await expect(variable.getByRole("button", { name: "save" })).toBeHidden()
	await variable.getByRole("button", { name: "Close details" }).click()

	// a dashboard with an svg widget: the editor lists the drawing's
	// binding once the file is picked
	await root.getByRole("button", { name: "Add", exact: true }).click()
	await page.getByRole("radio", { name: "Dashboard" }).check()
	await page.getByRole("button", { name: "add", exact: true }).click()
	const drow = tree.getByRole("treeitem", { name: "Dashboard: New, please edit" })
	const dkey = (await drow.getAttribute("data-key")) ?? ""
	const dashboard = page.locator(`[role="treeitem"][data-key="${dkey}"]`)
	await dashboard.getByRole("button", { name: "Open details" }).click()
	await dashboard.getByLabel("Description").fill("Lamp board")
	await dashboard.getByRole("button", { name: "save" }).click()
	await expect(dashboard.getByRole("button", { name: "save" })).toBeHidden()
	const href = (await dashboard.getByRole("link", { name: "open" }).getAttribute("href")) ?? ""

	await dashboard.getByRole("button", { name: "Add", exact: true }).click()
	await page.getByRole("radio", { name: "Widget" }).check()
	await page.getByRole("button", { name: "add", exact: true }).click()
	const wrow = tree.getByRole("treeitem", { name: "Widget: New, please edit" })
	const wkey = (await wrow.getAttribute("data-key")) ?? ""
	const widget = page.locator(`[role="treeitem"][data-key="${wkey}"]`)
	await widget.getByRole("button", { name: "Open details" }).click()
	await widget.getByLabel("Label").fill("Lamp")
	await widget.getByLabel("Component").selectOption("svg")
	const svgNode = widget.getByRole("combobox", { name: "svg node" })
	await svgNode.fill("lamp draw")
	await expect(page.getByRole("option", { name: /Lamp drawing/ }).first()).toBeVisible()
	await svgNode.press("ArrowDown")
	await svgNode.press("Enter")
	await expect(widget.getByRole("link", { name: "Lamp drawing" })).toBeVisible()
	const litNode = widget.getByRole("combobox", { name: "lit node" })
	await expect(litNode).toBeVisible()
	await litNode.fill("lamp swi")
	await expect(page.getByRole("option", { name: /Lamp switch/ }).first()).toBeVisible()
	await litNode.press("ArrowDown")
	await litNode.press("Enter")
	await expect(widget.getByRole("link", { name: "Lamp switch" })).toBeVisible()
	await widget.getByLabel("Height").fill("3")
	await widget.getByRole("button", { name: "save" }).click()
	await expect(widget.getByRole("button", { name: "save" })).toBeHidden()

	// the dashboard draws the lamp lit, and a click on the glass turns it off
	await page.goto(href)
	await expect(page.getByRole("heading", { name: "Lamp board" })).toBeVisible()
	const glass = page.locator(".glass[data-siot-toggle]")
	await expect(glass).toHaveAttribute("data-siot-state", "on")
	await glass.click()
	await expect(glass).toHaveAttribute("data-siot-state", "off")
	await glass.click()
	await expect(glass).toHaveAttribute("data-siot-state", "on")
})
