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
  CLIPBOARD: (message) => writeClipboard(message),
};

console.log("Simple IoT Javascript code");

var writeClipboard = (data) => {
  navigator.clipboard
    .writeText(data)
    .then(() => {
      console.log("copy success");
      // Success!
    })
    .catch((err) => {
      console.log("Something went wrong", err);
    });
};
