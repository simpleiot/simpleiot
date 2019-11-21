async function BleRequest() {
  let options = {
    acceptAllDevices: true
  };
  if (navigator.bluetooth == undefined) {
    alert("Sorry, Your device does not support Web BLE!");
    return;
  }

  let device = await navigator.bluetooth.requestDevice(options);
  if (!device) {
    throw "No device selected";
  }

  return device;
}
