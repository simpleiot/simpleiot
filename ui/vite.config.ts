import adapter from "@sveltejs/adapter-static"
import { sveltekit } from "@sveltejs/kit/vite"
import { defineConfig } from "vitest/config"

// The dev server proxies the API, the fonts, and the NATS WebSocket (which
// the siot HTTP server accepts at "/") to a running siot, so the app at
// http://localhost:5173/v2/ talks to it as the embedded build does. The
// port follows siot's own rule: SIOT_HTTP_PORT, or one above SIOT_NATS_PORT.
const httpPort =
	process.env.SIOT_HTTP_PORT ?? String(Number(process.env.SIOT_NATS_PORT ?? "4222") + 1)
const siot = `http://localhost:${httpPort}`

export default defineConfig({
	plugins: [
		sveltekit({
			adapter: adapter({ fallback: "index.html" }),
			paths: { base: "/v2", relative: false },
		}),
	],
	server: {
		proxy: {
			"/v1": siot,
			"/fonts": siot,
			"^/$": { target: siot, ws: true },
		},
		fs: { allow: ["../frontend/lib"] },
	},
	test: {
		include: ["src/**/*.test.ts"],
		passWithNoTests: true,
	},
})
