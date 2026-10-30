import { defineConfig } from "@playwright/test"

// The browser smoke test runs against a siot that siot_test_ui_browser
// starts with a fresh data directory; SIOT_E2E_URL is its HTTP address.
// SIOT_E2E_CHANNEL picks an installed browser, such as "chrome", instead
// of the one `npx playwright install chromium` downloads.
export default defineConfig({
	testDir: "e2e",
	timeout: 60_000,
	retries: 0,
	reporter: "list",
	use: {
		baseURL: process.env.SIOT_E2E_URL ?? "http://localhost:15223",
		channel: process.env.SIOT_E2E_CHANNEL,
		trace: "retain-on-failure",
	},
})
