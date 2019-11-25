const serviceUuid = "5c1b9a0d-b5be-4a40-8f7a-66b36d0a5176";
const charModelUuid = "fdcf0004-3fed-4ed2-84e6-04bbb9ae04d4";

export class BLE {
  constructor() {
    this.device = null;
    this.server = null;
  }

  async getState() {
    let ret = {
      connected: false,
      ssid: "",
      pass: "",
      model: ""
    };

    if (this.device && this.device.gatt.connected) {
      ret.connected = true;
    }

    if (!ret.connected) {
      // Nothing more to do
      return ret;
    }

    // Look up attributes
    const service = await this.server.getPrimaryService(serviceUuid);
    let characteristics = await service.getCharacteristics();
    console.log("characteristics: ", characteristics);

    let modelChar = await service.getCharacteristic(charModelUuid);
    let modelBuf = await modelChar.readValue();
    let decoder = new TextDecoder("utf-8");
    ret.model = decoder.decode(modelBuf);
    return ret;
  }

  async request() {
    let options = {
      acceptAllDevices: true,
      optionalServices: [serviceUuid]
    };
    if (navigator.bluetooth == undefined) {
      alert("Sorry, Your device does not support Web BLE!");
      return;
    }

    if (this.device && this.device.gatt.connected) {
      await this.device.gatt.disconnect();
    }

    this.device = await navigator.bluetooth.requestDevice(options);
    if (!this.device) {
      throw "No device selected";
    }

    this.server = await this.device.gatt.connect();

    return this.device;
  }

  async disconnect() {
    if (!this.device) {
      throw "no device";
    }

    if (!this.device.gatt.connected) {
      throw "not connected";
    }

    await this.device.gatt.disconnect();
  }
}
