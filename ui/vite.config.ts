import adapter from "@sveltejs/adapter-static"
import { sveltekit } from "@sveltejs/kit/vite"
import { defineConfig } from "vitest/config"

// The dev server proxies the API, the fonts, the Elm UI at /old, and the
// NATS WebSocket (which the siot HTTP server accepts at "/") to a running
// siot, so the app at http://localhost:5173/ talks to it as the embedded
// build does. The
// port follows siot's own rule: SIOT_HTTP_PORT, or one above SIOT_NATS_PORT.
const httpPort =
	process.env.SIOT_HTTP_PORT ?? String(Number(process.env.SIOT_NATS_PORT ?? "4222") + 1)
const siot = `http://localhost:${httpPort}`

export default defineConfig({
	plugins: [
		sveltekit({
			adapter: adapter({ fallback: "index.html" }),
			paths: { relative: false },
		}),
	],
	server: {
		proxy: {
			"/v1": siot,
			"/fonts": siot,
			"/old": siot,
			// only the WebSocket; the page at "/" is the dev server's
			"^/$": {
				target: siot,
				ws: true,
				bypass: (req) => (req.headers.upgrade ? undefined : req.url),
			},
		},
		fs: { allow: ["../frontend/lib"] },
	},
	test: {
		include: ["src/**/*.test.ts"],
	},
})
