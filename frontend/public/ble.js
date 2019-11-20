async function BleRequest() {
  if (navigator.bluetooth == undefined) {
    alert("Sorry, Your device does not support Web BLE!");
    return;
  }

  console.log("requesting BLE devices ...");
  var device = await navigator.bluetooth.requestDevice();
  console.log("device: ", device);
}
