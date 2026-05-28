// Periodic dashboard reload, controlled by a <select> in the page header.
// Selection persists in localStorage so it survives the reload it triggers.
(function () {
  const STORAGE_KEY = "goldilocks:autoRefreshSeconds";
  const select = document.getElementById("js-autorefresh");
  if (!select) return;

  let timer = null;

  function schedule(seconds) {
    if (timer) {
      clearTimeout(timer);
      timer = null;
    }
    if (seconds > 0) {
      timer = setTimeout(() => window.location.reload(), seconds * 1000);
    }
  }

  function persist(seconds) {
    try {
      if (seconds > 0) localStorage.setItem(STORAGE_KEY, String(seconds));
      else localStorage.removeItem(STORAGE_KEY);
    } catch (_) {
      // ignore (private mode etc.)
    }
  }

  // Restore previous selection.
  let restored = 0;
  try {
    restored = parseInt(localStorage.getItem(STORAGE_KEY) || "0", 10) || 0;
  } catch (_) {}
  if (restored > 0) {
    const opt = Array.from(select.options).find(
      (o) => parseInt(o.value, 10) === restored
    );
    if (opt) {
      select.value = String(restored);
      schedule(restored);
    }
  }

  select.addEventListener("change", () => {
    const seconds = parseInt(select.value, 10) || 0;
    persist(seconds);
    schedule(seconds);
  });
})();
