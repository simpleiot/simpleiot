/*
 * Simple IoT documentation theme behavior.
 *
 * mdBook renders the book title in the menu bar as plain text. This turns it
 * into a link back to the main site at simpleiot.org, which is the usual place
 * a reader expects the name at the top of a page to take them. Loaded through
 * `additional-js` in book.toml so the built-in templates stay unmodified.
 */

(function () {
  "use strict";

  var HOME_URL = "https://simpleiot.org";

  function linkMenuTitle() {
    var title = document.querySelector(".menu-title");
    if (!title || title.querySelector("a")) {
      return;
    }

    var link = document.createElement("a");
    link.className = "menu-title-link";
    link.href = HOME_URL;
    link.title = "Simple IoT home";
    link.textContent = title.textContent.trim();

    title.textContent = "";
    title.appendChild(link);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", linkMenuTitle);
  } else {
    linkMenuTitle();
  }
})();
