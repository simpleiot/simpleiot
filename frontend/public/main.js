import {BLE} from "./ble.js";

export const main = (app) => {
  var ble = new BLE();

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

  app.ports.portOut.subscribe(async function(data) {
    console.log("portOut message: ", data);
    let state;
    switch (data.cmd) {
      case "scan":
        try {
          let d = await ble.request();
          console.log("device selected: ", d);
        } catch (e) {
          console.log("scanning error: ", e);
        }
        state = await ble.getState();
        app.ports.portIn.send(state);
        break;
      case "disconnect":
        try {
          await ble.disconnect();
        } catch (e) {
          console.log("disconnect error: ", e);
        }
        state = await ble.getState();
        app.ports.portIn.send(state);
        break;
      default:
        console.log("unknown cmd: ", data.cmd);
    }
  });
};
