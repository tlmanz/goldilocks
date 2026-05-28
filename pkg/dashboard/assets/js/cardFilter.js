// Click-to-filter for the top-of-page summary cards.
// Sets body[data-state-filter] which CSS uses to hide non-matching containers
// (and their now-empty parent workloads/namespaces) via :has().
//
// Active state is tracked by the *clicked card identity* (not by the filter
// value) so that two cards which happen to share the same filter — e.g.
// "Over-provisioned" and "Potential savings" both target state="over" — don't
// both light up when only one was clicked.
(function () {
  const cards = Array.from(document.querySelectorAll(".js-card-filter"));
  if (!cards.length) return;

  // The "reset" card (Namespaces) has an empty data-state-filter and acts as
  // the no-filter-active indicator.
  const resetCard = cards.find((c) => !(c.dataset.stateFilter || "")) || null;

  function setActive(activeCard) {
    cards.forEach((c) => {
      c.setAttribute("aria-pressed", c === activeCard ? "true" : "false");
    });

    const filterValue = activeCard ? (activeCard.dataset.stateFilter || "") : "";
    if (filterValue) {
      document.body.dataset.stateFilter = filterValue;
    } else {
      delete document.body.dataset.stateFilter;
    }
  }

  cards.forEach((card) => {
    card.addEventListener("click", function () {
      const own = card.dataset.stateFilter || "";
      const isPressed = card.getAttribute("aria-pressed") === "true";
      // Clicking the reset card OR re-clicking the active card returns to
      // the "no filter" state (with the reset card visually pressed).
      if (!own || isPressed) {
        setActive(resetCard);
      } else {
        setActive(card);
      }
    });
  });
})();
