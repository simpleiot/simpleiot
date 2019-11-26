import {BLE} from "./ble.js";

export const main = (app) => {
  var ble = new BLE(async () => {
    let state = await ble.getState();
    app.ports.portIn.send(state);
  });

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
    let state;
    switch (data.cmd) {
      case "scan":
        try {
          await ble.request();
          state = await ble.getState();
          app.ports.portIn.send(state);
        } catch (e) {
          console.log("scanning error: ", e);
        }
        break;
      case "disconnect":
        try {
          await ble.disconnect();
          state = await ble.getState();
          app.ports.portIn.send(state);
        } catch (e) {
          console.log("disconnect error: ", e);
        }
        break;
      case "configureGw":
        try {
          await ble.configureGw(data);
          state = await ble.getState();
          app.ports.portIn.send(state);
        } catch (e) {
          console.log("configure GW error: ", e);
        }
        break;

      default:
        console.log("unknown cmd: ", data.cmd);
    }
  });
};
