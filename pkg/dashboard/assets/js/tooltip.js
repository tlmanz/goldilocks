// Lightweight click-toggle tooltip for elements with [data-tooltip].
// Falls back to native title= behavior on hover (set via aria-describedby).
(function () {
  const OPEN_CLASS = "tooltip--open";
  let openEl = null;

  function close() {
    if (!openEl) return;
    openEl.classList.remove(OPEN_CLASS);
    openEl = null;
  }

  document.addEventListener("click", function (event) {
    const trigger = event.target.closest("[data-tooltip]");
    if (!trigger) {
      close();
      return;
    }
    event.preventDefault();
    event.stopPropagation();
    if (openEl === trigger) {
      close();
      return;
    }
    close();
    trigger.classList.add(OPEN_CLASS);
    openEl = trigger;
  });

  document.addEventListener("keydown", function (event) {
    if (event.key === "Escape") close();
  });
})();
