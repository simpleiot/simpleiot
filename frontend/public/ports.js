// On load, listen to Elm!
window.addEventListener("load", (_) => {
  window.ports = {
    init: (app) =>
      app.ports.out.subscribe(({ action, data }) =>
        actions[action]
          ? actions[action](data)
          : console.warn(`I didn't recognize action "${action}".`)
      ),
  };
});

// maps actions to functions!
const actions = {
  LOG: (message) => console.log(`From Elm:`, message),
  CLIPBOARD: (message) => {
    if (navigator.clipboard) {
      writeClipboard(message);
    } else {
      console.log("clipboard not available");
    }
  },
};

console.log("Simple IoT Javascript code");

var writeClipboard = (data) => {
  navigator.clipboard
    .writeText(data)
    .then(() => {
      // FIXME, should probably send something back to elm
      // Success!
    })
    .catch((err) => {
      console.log("Something went wrong", err);
    });
};
