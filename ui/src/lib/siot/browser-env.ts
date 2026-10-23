// The browser surroundings for the NATS connection.

import { connect, ErrorCode } from "simpleiot-js"
import type { Env } from "./connection.ts"

function wsUrl() {
	const proto = location.protocol === "https:" ? "wss:" : "ws:"
	return `${proto}//${location.host}/`
}

const authErrors: unknown[] = [
	ErrorCode.AuthorizationViolation,
	ErrorCode.AuthenticationExpired,
	ErrorCode.AuthenticationTimeout,
]

export const browserEnv: Env = {
	connect: (token) => connect({ url: wsUrl(), token }),
	isAuthError: (err) =>
		typeof err === "object" && err !== null && "code" in err && authErrors.includes(err.code),
	hidden: () => document.hidden,
	onVisibilityChange(f) {
		document.addEventListener("visibilitychange", f)
		return () => {
			document.removeEventListener("visibilitychange", f)
		}
	},
	frame(f) {
		if (document.hidden) {
			setTimeout(f, 250)
		} else {
			requestAnimationFrame(f)
		}
	},
	setTimeout: (f, ms) => setTimeout(f, ms),
	clearTimeout: (t) => {
		clearTimeout(t as ReturnType<typeof setTimeout>)
	},
	warn: (...args) => {
		console.warn(...args)
	},
}
