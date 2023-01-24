// import { BLE } from "./ble.js"

// eslint-disable-next-line no-undef
const app = Elm.Main.init({
	flags: JSON.parse(localStorage.getItem("storage")),
})

app.ports.save_.subscribe((storage) => {
	localStorage.setItem("storage", JSON.stringify(storage))
	app.ports.load_.send(storage)
})

app.ports.out.subscribe(({ action, data }) =>
	actions[action]
		? actions[action](data)
		: console.warn(`I didn't recognize action "${action}".`)
)

// maps actions to functions!
const actions = {
	LOG: (message) => console.log(`From Elm:`, message),
	CLIPBOARD: (message) => {
		if (navigator.clipboard) {
			writeClipboard(message)
		} else {
			console.log("clipboard not available")
		}
	},
}

console.log("Simple IoT Javascript code")

const writeClipboard = (data) => {
	navigator.clipboard
		.writeText(data)
		.then(() => {
			// FIXME, should probably send something back to elm
			// Success!
		})
		.catch((err) => {
			console.log("Something went wrong", err)
		})
}

/*
export const main = (app) => {
	const ble = new BLE(async () => {
		const state = await ble.getState()
		app.ports.portIn.send(state)
	})
  */
/*
 * Websocket code needs cleaned up
 *
 * Var loc = window.location,
 * new_uri;
 * if (loc.protocol === "https:") {
 * ws_uri = "wss:";
 * } else {
 * ws_uri = "ws:";
 * }
 * ws_uri += "//" + loc.host;
 * ws_uri += "/ws";
 * var conn = new WebSocket(ws_uri);
 * conn.onclose = function(evt) {
 * console.log("WS connection closed");
 * };
 * conn.onmessage = function(evt) {
 * var obj = JSON.parse(evt.data);
 * app.ports.portIn.send(obj);
 * };
 */
/*

	app.ports.portOut.subscribe(async function (data) {
		let state
		switch (data.cmd) {
			case "scan":
				try {
					await ble.request()
					state = await ble.getState()
					app.ports.portIn.send(state)
				} catch (e) {
					console.log("scanning error: ", e)
				}
				break
			case "disconnect":
				try {
					await ble.disconnect()
					state = await ble.getState()
					app.ports.portIn.send(state)
				} catch (e) {
					console.log("disconnect error: ", e)
				}
				break
			case "configureWifi":
				try {
					await ble.configureWifi(data)
					state = await ble.getState()
					app.ports.portIn.send(state)
				} catch (e) {
					console.log("configure GW WiFi error: ", e)
				}
				break
			case "configureTimer":
				try {
					await ble.configureTimer(data)
					state = await ble.getState()
					app.ports.portIn.send(state)
				} catch (e) {
					console.log("configure GW configure Timer error: ", e)
				}
				break

			case "fireTimer":
				try {
					await ble.fireTimer(data)
					state = await ble.getState()
					app.ports.portIn.send(state)
				} catch (e) {
					console.log("configure GW fire Timer error: ", e)
				}
				break

			default:
				console.log("unknown cmd: ", data.cmd)
		}
	})
}
  */
