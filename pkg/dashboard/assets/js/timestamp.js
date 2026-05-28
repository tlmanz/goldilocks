// Renders a friendly "X minutes ago" label on any <time data-unix=...> element.
(function () {
  function formatRelative(unixSeconds) {
    const ageMs = Date.now() - unixSeconds * 1000;
    const ageSec = Math.max(0, Math.round(ageMs / 1000));
    if (ageSec < 30) return "just now";
    if (ageSec < 60) return `${ageSec}s ago`;
    if (ageSec < 3600) return `${Math.round(ageSec / 60)} min ago`;
    if (ageSec < 86400) return `${Math.round(ageSec / 3600)}h ago`;
    return `${Math.round(ageSec / 86400)}d ago`;
  }

  function refresh() {
    document.querySelectorAll("time[data-unix]").forEach((el) => {
      const unix = parseInt(el.dataset.unix, 10);
      if (isNaN(unix)) return;
      el.textContent = formatRelative(unix);
      const iso = new Date(unix * 1000).toLocaleString();
      el.setAttribute("datetime", new Date(unix * 1000).toISOString());
      el.setAttribute("title", iso);
    });
  }

  refresh();
  setInterval(refresh, 30000);
})();
