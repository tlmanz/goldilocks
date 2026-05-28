// "Mark as reviewed" — per-namespace and per-workload, persisted in localStorage.
// A "Show reviewed" toggle in the page header reveals dimmed reviewed entries.
(function () {
  const STORAGE_KEY = "goldilocks:reviewedIds";
  const SHOW_KEY = "goldilocks:showReviewed";
  const REVIEW_CLASS = "is-reviewed";
  const SHOW_REVIEWED_CLASS = "show-reviewed";

  function loadSet(key) {
    try {
      const raw = localStorage.getItem(key);
      if (!raw) return new Set();
      return new Set(JSON.parse(raw));
    } catch (_) {
      return new Set();
    }
  }

  function saveSet(key, set) {
    try {
      localStorage.setItem(key, JSON.stringify(Array.from(set)));
    } catch (_) {}
  }

  const reviewed = loadSet(STORAGE_KEY);

  function applyReviewed() {
    document.querySelectorAll("[data-review-id]").forEach((el) => {
      if (reviewed.has(el.dataset.reviewId)) {
        el.classList.add(REVIEW_CLASS);
      } else {
        el.classList.remove(REVIEW_CLASS);
      }
    });
    document.querySelectorAll(".js-review").forEach((btn) => {
      const sel = btn.dataset.reviewTarget;
      if (!sel) return;
      const target = document.querySelector(sel);
      if (!target) return;
      const isReviewed = reviewed.has(target.dataset.reviewId);
      const label = btn.querySelector(".js-review-label");
      if (label) label.textContent = isReviewed ? "Reviewed" : "Mark reviewed";
      btn.setAttribute("aria-pressed", isReviewed ? "true" : "false");
    });
  }

  document.addEventListener("click", function (event) {
    const button = event.target.closest(".js-review");
    if (!button) return;
    event.preventDefault();
    const sel = button.dataset.reviewTarget;
    if (!sel) return;
    const target = document.querySelector(sel);
    if (!target) return;
    const id = target.dataset.reviewId;
    if (!id) return;

    if (reviewed.has(id)) {
      reviewed.delete(id);
    } else {
      reviewed.add(id);
    }
    saveSet(STORAGE_KEY, reviewed);
    applyReviewed();
  });

  // "Show reviewed" toggle on the page header.
  const showBtn = document.querySelector(".js-show-reviewed");
  function restoreShowReviewed() {
    let value = false;
    try {
      value = localStorage.getItem(SHOW_KEY) === "true";
    } catch (_) {}
    document.body.classList.toggle(SHOW_REVIEWED_CLASS, value);
    if (showBtn) {
      showBtn.setAttribute("aria-pressed", value ? "true" : "false");
      const label = showBtn.querySelector(".js-show-reviewed-label");
      if (label) label.textContent = value ? "Hide reviewed" : "Show reviewed";
    }
  }

  if (showBtn) {
    showBtn.addEventListener("click", function () {
      const next = !document.body.classList.contains(SHOW_REVIEWED_CLASS);
      document.body.classList.toggle(SHOW_REVIEWED_CLASS, next);
      try {
        localStorage.setItem(SHOW_KEY, next ? "true" : "false");
      } catch (_) {}
      restoreShowReviewed();
    });
  }

  applyReviewed();
  restoreShowReviewed();
})();
