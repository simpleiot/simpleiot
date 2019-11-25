export class BLE {
  constructor() {
    this.device = null;
  }

  getState() {
    let ret = {
      connected: false,
      ssid: "",
      pass: ""
    };

    if (this.device && this.device.gatt.connected) {
      ret.connected = true;
    }

    return ret;
  }

  async request() {
    let options = {
      acceptAllDevices: true
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

    await this.device.gatt.connect();

    return this.device;
  }

  async connect() {
    if (!this.device) {
      return Promise.reject("no device");
    }
    await this.device.gatt.connect();
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
