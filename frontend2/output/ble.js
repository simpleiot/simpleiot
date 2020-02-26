const serviceUuid = "5c1b9a0d-b5be-4a40-8f7a-66b36d0a5176";
const charUptimeUuid = "fdcf0000-3fed-4ed2-84e6-04bbb9ae04d4";
const charSignalUuid = "fdcf0001-3fed-4ed2-84e6-04bbb9ae04d4";
const charFreeMemUuid = "fdcf0002-3fed-4ed2-84e6-04bbb9ae04d4";
const charModelUuid = "fdcf0003-3fed-4ed2-84e6-04bbb9ae04d4";
const charWifiSSIDUuid = "fdcf0004-3fed-4ed2-84e6-04bbb9ae04d4";
const charConnectedUuid = "fdcf0005-3fed-4ed2-84e6-04bbb9ae04d4";
const charSetWifiSSIDUuid = "fdcf0006-3fed-4ed2-84e6-04bbb9ae04d4";
const charSetWifiPassUuid = "fdcf0007-3fed-4ed2-84e6-04bbb9ae04d4";
// Const charDeviceNameUuid = "fdcf0008-3fed-4ed2-84e6-04bbb9ae04d4";
const charTimerFireDurationUuid = "fdcf0009-3fed-4ed2-84e6-04bbb9ae04d4";
const charTimerFireUuid = "fdcf000a-3fed-4ed2-84e6-04bbb9ae04d4";
const charCurrentTimeUuid = "fdcf000b-3fed-4ed2-84e6-04bbb9ae04d4";
const charTimerFireTimeUuid = "fdcf000c-3fed-4ed2-84e6-04bbb9ae04d4";
const charTimerFireCountUuid = "fdcf000d-3fed-4ed2-84e6-04bbb9ae04d4";

export class BLE {
  constructor(stateChanged) {
    this.resetState();
    this.stateChanged = stateChanged;
  }

  resetState() {
    this.device = null;
    this.server = null;
    this.service = null;
    this.uptime = 0;
    this.signal = 0;
    this.freeMem = 0;
    this.connected = false;
    this.currentTime = 0;
    this.timerFireCount = 0;
  }

  onDisconnected() {
    this.resetState();
    this.stateChanged();
  }

  onCurrentTimeChanged(event) {
    let {value} = event.target;
    this.currentTime = value.getUint32();
    this.stateChanged();
  }

  onConnectedChanged(event) {
    let {value} = event.target;
    this.connected = value.getUint8();
    if (this.connected) {
      this.connected = true;
    } else {
      this.connected = false;
    }
    console.log("onConnectedChanged: ", value.getUint8());
    this.stateChanged();
  }

  onUptimeChanged(event) {
    let {value} = event.target;
    this.uptime = value.getInt32();
  }

  onSignalChanged(event) {
    let {value} = event.target;
    this.signal = value.getUint8();
  }

  onFreeMemChanged(event) {
    let {value} = event.target;
    this.freeMem = value.getInt32();
    this.stateChanged();
  }

  onTimerFireCountChanged(event) {
    let {value} = event.target;
    this.timerFireCount = value.getInt32();
    this.stateChanged();
  }

  async configureWifi(config) {
    if (!this.device) {
      throw "configure GW, no device";
    }

    const charSetWifiSSID = await this.service.getCharacteristic(charSetWifiSSIDUuid);
    const encoder = new TextEncoder();
    charSetWifiSSID.writeValue(encoder.encode(config.wifiSSID));

    const charSetWifiPass = await this.service.getCharacteristic(charSetWifiPassUuid);
    charSetWifiPass.writeValue(encoder.encode(config.wifiPass));
  }

  async configureTimer(config) {
    if (!this.device) {
      throw "configure GW timer, no device";
    }

    const charTimerFireDuration = await this.service.getCharacteristic(charTimerFireDurationUuid);
    charTimerFireDuration.writeValue(Int32Array.of(config.fireDuration));

    const charCurrentTime = await this.service.getCharacteristic(charCurrentTimeUuid);
    let d = new Date();
    let n = d.getTime();
    let k = 1000;
    charCurrentTime.writeValue(Int32Array.of(n / k));

    let parts = config.fireTime.split(":");
    if (parts.length < 2) {
      console.log("Error parsing time");
      return;
    }

    let fireTimeMin =
      Number(parts[0]) * 60 + Number(parts[1]) + d.getTimezoneOffset();

    if (fireTimeMin >= 60 * 24) {
      // Roll over into next day
      fireTimeMin -= 60 * 24;
    }
    // The above is localtime, so we need to convert to UTC
    let charTimerFireTime = await this.service.getCharacteristic(charTimerFireTimeUuid);
    charTimerFireTime.writeValue(Int32Array.of(fireTimeMin));
  }

  async fireTimer() {
    if (!this.device) {
      throw "fire timer, no device";
    }

    const charSetWifiSSID = await this.service.getCharacteristic(charTimerFireUuid);
    let zero = 0;
    charSetWifiSSID.writeValue(Uint8Array.of(zero));
  }

  async getState() {
    console.log("getState: service: ", this.service);
    let ret = {
      connected: this.connected,
      bleConnected: false,
      ssid: "",
      pass: "",
      model: "",
      uptime: this.uptime,
      signal: this.signal,
      freeMem: this.freeMem,
      currentTime: this.currentTime,
      timerFireDuration: 0,
      timerFireTime: 0,
      timerFireCount: this.timerFireCount
    };

    if (this.device && this.device.gatt.connected) {
      ret.bleConnected = true;
    }

    if (!ret.bleConnected) {
      // Nothing more to do
      return ret;
    }

    const modelChar = await this.service.getCharacteristic(charModelUuid);
    let buf = await modelChar.readValue();
    const decoder = new TextDecoder("utf-8");
    ret.model = decoder.decode(buf);

    const ssidChar = await this.service.getCharacteristic(charWifiSSIDUuid);
    buf = await ssidChar.readValue();
    ret.ssid = decoder.decode(buf);

    const fireDurChar = await this.service.getCharacteristic(charTimerFireDurationUuid);
    buf = await fireDurChar.readValue();
    ret.timerFireDuration = buf.getUint32();

    const fireTimeChar = await this.service.getCharacteristic(charTimerFireTimeUuid);
    buf = await fireTimeChar.readValue();
    ret.timerFireTime = buf.getUint32();

    let d = new Date();
    ret.timerFireTime -= d.getTimezoneOffset();
    if (ret.timerFireTime < 0) {
      ret.timerFireTime += 24 * 60;
    }

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

    try {
      if (this.device && this.device.gatt.connected) {
        await this.device.gatt.disconnect();
      }

      this.device = await navigator.bluetooth.requestDevice(options);
      if (!this.device) {
        throw "No device selected";
      }

      this.device.addEventListener(
        "gattserverdisconnected",
        this.onDisconnected.bind(this)
      );

      this.server = await this.device.gatt.connect();
      this.service = await this.server.getPrimaryService(serviceUuid);

      console.log("CLIFF: got service");

      const connectedChar = await this.service.getCharacteristic(charConnectedUuid);
      let buf = await connectedChar.readValue();
      this.connected = buf.getUint8();

      if (this.connected) {
        this.connected = true;
      } else {
        this.connected = false;
      }

      const connectChar = await this.service.getCharacteristic(charConnectedUuid);
      await connectChar.startNotifications();
      connectChar.addEventListener(
        "characteristicvaluechanged",
        this.onConnectedChanged.bind(this)
      );

      /*
       * Const uptimeChar = await this.service.getCharacteristic(charUptimeUuid);
       * await uptimeChar.startNotifications();
       * uptimeChar.addEventListener(
       * "characteristicvaluechanged",
       * this.onUptimeChanged.bind(this)
       * );
       */

      const signalChar = await this.service.getCharacteristic(charSignalUuid);
      await signalChar.startNotifications();
      signalChar.addEventListener(
        "characteristicvaluechanged",
        this.onSignalChanged.bind(this)
      );

      const freeMemChar = await this.service.getCharacteristic(charFreeMemUuid);
      await freeMemChar.startNotifications();
      freeMemChar.addEventListener(
        "characteristicvaluechanged",
        this.onFreeMemChanged.bind(this)
      );

      try {
        const timerFireCountChar = await this.service.getCharacteristic(charTimerFireCountUuid);
        await timerFireCountChar.startNotifications();
        timerFireCountChar.addEventListener(
          "characteristicvaluechanged",
          this.onTimerFireCountChanged.bind(this)
        );

        buf = await timerFireCountChar.readValue();
        this.timerFireCount = buf.getInt32();
      } catch (e) {
        console.log("Error getting timerCountChar");
      }

      const currentTimeChar = await this.service.getCharacteristic(charCurrentTimeUuid);
      await currentTimeChar.startNotifications();
      currentTimeChar.addEventListener(
        "characteristicvaluechanged",
        this.onCurrentTimeChanged.bind(this)
      );
    } catch (e) {
      console.log("Error connecting: ", e);
      this.resetState();
      throw e;
    }

    return this.device;
  }

  async disconnect() {
    if (!this.device) {
      this.resetState();
      throw "no device";
    }

    if (!this.device.gatt.connected) {
      this.resetState();
      throw "not connected";
    }

    try {
      await this.device.gatt.disconnect();
    } catch (e) {
      console.log("Error disconnecting: ", e);
    } finally {
      this.resetState();
    }
  }
}
