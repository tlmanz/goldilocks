// Sort namespace entries (and their workload children, when present) by user selection.
// Supports both the dashboard page (<article data-namespace>) and the namespace
// list page (<li data-filter>).
(function () {
  const select = document.getElementById("js-sort-select");
  const container = document.getElementById("js-filter-container");
  if (!select || !container) return;

  function getEntries() {
    let entries = Array.from(container.querySelectorAll("article[data-namespace]"));
    if (entries.length === 0) {
      entries = Array.from(container.querySelectorAll(":scope > [data-filter]"));
    }
    return entries;
  }

  function entryName(el) {
    return (
      el.dataset.namespace ||
      el.dataset.filter ||
      el.textContent ||
      ""
    ).trim();
  }

  function getDeviationScore(el) {
    const bars = el.querySelectorAll(".diffBar");
    let total = 0;
    bars.forEach((b) => {
      const m = (b.getAttribute("style") || "").match(/--diff-pct:\s*(\d+)/);
      if (m) total += parseInt(m[1], 10);
    });
    return total;
  }

  function compareName(a, b) {
    return entryName(a).localeCompare(entryName(b));
  }

  function compareDeviation(a, b) {
    return getDeviationScore(b) - getDeviationScore(a);
  }

  function getSavingsScore(el) {
    return parseInt(el.dataset.savings || "0", 10) || 0;
  }

  function compareSavings(a, b) {
    return getSavingsScore(b) - getSavingsScore(a);
  }

  function workloadCount(el) {
    const attr = parseInt(el.dataset.workloadCount || "", 10);
    if (!isNaN(attr)) return attr;
    return el.querySelectorAll(".js-workload").length;
  }

  function compareWorkloadCount(a, b) {
    return workloadCount(b) - workloadCount(a);
  }

  function attentionCount(el) {
    return parseInt(el.dataset.needsAttention || "0", 10) || 0;
  }

  function compareAttention(a, b) {
    return attentionCount(b) - attentionCount(a);
  }

  function sortEntries(mode) {
    const entries = getEntries();
    if (entries.length === 0) return;

    let cmp;
    switch (mode) {
      case "savings":
        cmp = compareSavings;
        break;
      case "deviation":
        // On the namespace list page, no .diffBar children exist; fall back
        // to needs-attention count as a proxy for "deviation magnitude".
        cmp = (a, b) => {
          const d = compareDeviation(a, b);
          return d !== 0 ? d : compareAttention(a, b);
        };
        break;
      case "workloads":
        cmp = compareWorkloadCount;
        break;
      case "name":
      default:
        cmp = compareName;
    }
    entries.sort(cmp).forEach((a) => container.appendChild(a));

    if (mode === "deviation") {
      entries.forEach((entry) => {
        const wls = Array.from(entry.querySelectorAll(".js-workload"));
        wls
          .sort((a, b) => getDeviationScore(b) - getDeviationScore(a))
          .forEach((w) => w.parentElement.appendChild(w));
      });
    }
  }

  // Sort options that depend on per-namespace stats (workloads / savings /
  // deviation) only make sense if at least one entry has that data attribute.
  // The namespace list page now has these attrs when the cache provides
  // summary data; remove them only if literally no entry exposes them.
  const sample = container.querySelector("[data-namespace], [data-filter]");
  if (sample && !sample.dataset.savings && !sample.dataset.workloadCount && !sample.querySelector(".js-workload")) {
    Array.from(select.options).forEach((opt) => {
      if (
        opt.value === "deviation" ||
        opt.value === "workloads" ||
        opt.value === "savings"
      ) {
        opt.remove();
      }
    });
  }

  select.addEventListener("change", () => sortEntries(select.value));

  if (select.value && select.value !== "name") {
    sortEntries(select.value);
  }
})();
