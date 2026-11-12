import { expect, test, type Page } from "@playwright/test"

// A one-pixel PNG, as a binary upload stores it.
const png = Buffer.from(
	"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg==",
	"base64"
)

const lamp = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 130">
  <style>.glass[data-siot-state="on"] { fill: #ffd54a; }</style>
  <circle class="glass" cx="50" cy="48" r="34" data-siot-bind="lit" data-siot-toggle />
</svg>`

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

// uploadBinary uploads through the Upload button, which is what carries
// the Binary setting to the upload
async function uploadBinary(
	page: Page,
	node: ReturnType<Page["locator"]>,
	name: string,
	buffer: Buffer
) {
	const chooser = page.waitForEvent("filechooser")
	await node.getByRole("button", { name: "Upload new file" }).click()
	await (await chooser).setFiles({ name, mimeType: "image/png", buffer })
}

async function save(node: ReturnType<Page["locator"]>) {
	await node.getByRole("button", { name: "save" }).click()
	await expect(node.getByRole("button", { name: "save" })).toBeHidden()
}

test("file nodes preview an image and exercise a drawing", async ({ page }) => {
	const { root } = await signIn(page)

	// a photo uploaded as binary shows in the node's settings
	const { node: photo } = await add(page, root, "File")
	await photo.getByLabel("Description").fill("Preview photo")
	await photo.getByRole("checkbox", { name: "Binary" }).check()
	await save(photo)
	await uploadBinary(page, photo, "photo.png", png)
	await expect(photo.getByRole("img", { name: "Preview photo" })).toBeVisible()
	await photo.getByRole("button", { name: "Close details" }).click()

	// a drawing shows with a test value for its binding, which a typed
	// value and a click on the drawing both change
	const { node: drawing } = await add(page, root, "File")
	await drawing.getByLabel("Description").fill("Preview lamp")
	await save(drawing)
	await drawing.locator('input[type="file"]').setInputFiles({
		name: "lamp.svg",
		mimeType: "image/svg+xml",
		buffer: Buffer.from(lamp),
	})
	const glass = drawing.locator(".glass")
	await expect(glass).toHaveAttribute("data-siot-state", "stale")
	const lit = drawing.getByLabel("lit:", { exact: true })
	await lit.fill("1")
	await expect(glass).toHaveAttribute("data-siot-state", "on")
	await glass.click()
	await expect(glass).toHaveAttribute("data-siot-state", "off")
	await expect(lit).toHaveValue("0")
	// nothing was sent, so the node has no unsaved edits
	await expect(drawing.getByRole("button", { name: "save" })).toBeHidden()
})

test("an image widget on a canvas moves through the stack", async ({ page }) => {
	const { root } = await signIn(page)

	const { node: photo } = await add(page, root, "File")
	await photo.getByLabel("Description").fill("Stack photo")
	await photo.getByRole("checkbox", { name: "Binary" }).check()
	await save(photo)
	await uploadBinary(page, photo, "photo.png", png)
	await expect(photo.getByRole("img", { name: "Stack photo" })).toBeVisible()
	await photo.getByRole("button", { name: "Close details" }).click()

	const { node: dashboard } = await add(page, root, "Dashboard")
	await dashboard.getByLabel("Description").fill("Stack board")
	await dashboard.getByRole("radio", { name: "canvas" }).check()
	await save(dashboard)
	const href = (await dashboard.getByRole("link", { name: "open" }).getAttribute("href")) ?? ""

	// an image and a value tile, neither with a Z
	const { node: image } = await add(page, dashboard, "Widget")
	await image.getByLabel("Label").fill("Site")
	await image.getByLabel("Component").selectOption("image")
	const imageNode = image.getByRole("combobox", { name: "image node" })
	await imageNode.fill("stack ph")
	await expect(page.getByRole("option", { name: /Stack photo/ }).first()).toBeVisible()
	await imageNode.press("ArrowDown")
	await imageNode.press("Enter")
	await expect(image.getByRole("link", { name: "Stack photo" })).toBeVisible()
	await image.getByLabel("X:", { exact: true }).fill("20")
	await image.getByLabel("Y:", { exact: true }).fill("20")
	await save(image)
	await image.getByRole("button", { name: "Close details" }).click()

	const { node: tile } = await add(page, dashboard, "Widget")
	await tile.getByLabel("Label").fill("Reading")
	await tile.getByLabel("Component").selectOption("value")
	await tile.getByLabel("X:", { exact: true }).fill("500")
	await tile.getByLabel("Y:", { exact: true }).fill("20")
	await save(tile)

	// the image draws, and the stacking buttons move it to the top and the
	// bottom of the drawing order
	await page.setViewportSize({ width: 1200, height: 800 })
	await page.goto(href)
	await expect(page.getByRole("heading", { name: "Stack board" })).toBeVisible()
	const img = page.getByRole("img", { name: "Site" })
	await expect(img).toBeVisible()
	await page.getByRole("button", { name: "edit" }).click()
	await page.locator(".equipment", { has: img }).click()
	const equipment = page.locator(".plane > .equipment")
	await page.getByRole("button", { name: "bring to front" }).click()
	await expect(equipment.last().getByRole("img", { name: "Site" })).toBeVisible()
	await page.getByRole("button", { name: "send to back" }).click()
	await expect(equipment.first().getByRole("img", { name: "Site" })).toBeVisible()
})
