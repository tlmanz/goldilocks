// Trend visualization for each container. Lazy-loads per-container time
// series from /api/history via IntersectionObserver (so a dashboard with
// 500 containers doesn't fire 500 requests at once), renders a small
// sparkline inline, and opens a fuller chart in a <dialog> on click.
//
// If the dashboard wasn't started with --history-db the endpoint returns
// 503 — we render a quiet "history not configured" placeholder.
(function () {
  const BASE = (document.querySelector("base")?.getAttribute("href") || "/").replace(/\/$/, "");
  const HOURS_DEFAULT = 24;
  const HOURS_EXPANDED = 168;
  const SPARK_WIDTH = 220;
  const SPARK_HEIGHT = 44;
  const CHART_WIDTH = 720;
  const CHART_HEIGHT = 280;

  const cards = document.querySelectorAll(".js-history");
  if (!cards.length) return;

  // Shared dialog reused for every "Expand" click.
  const dialog = ensureDialog();
  let historyEnabled = true;

  async function fetchSeries(card, hours) {
    const url = `${BASE}/api/history?ns=${encodeURIComponent(card.dataset.ns)}&kind=${encodeURIComponent(card.dataset.kind)}&workload=${encodeURIComponent(card.dataset.workload)}&container=${encodeURIComponent(card.dataset.container)}&hours=${hours}`;
    const res = await fetch(url);
    if (res.status === 503) {
      historyEnabled = false;
      throw new Error("history not configured");
    }
    if (!res.ok) {
      throw new Error(`history fetch failed: ${res.status}`);
    }
    return res.json();
  }

  function renderSpark(card, payload) {
    const body = card.querySelector(".historyCard__body");
    const btn = card.querySelector(".js-history-expand");
    if (!payload.points || payload.points.length === 0) {
      body.innerHTML = '<span class="historyCard__placeholder">No history yet — the collector will fill this in over the next few intervals.</span>';
      return;
    }
    body.innerHTML = "";
    body.appendChild(renderSparkSVG(payload.points));
    if (btn) {
      btn.disabled = false;
      btn.addEventListener("click", () => openExpanded(card));
    }
  }

  function renderError(card, message) {
    const body = card.querySelector(".historyCard__body");
    body.innerHTML = `<span class="historyCard__placeholder historyCard__placeholder--muted">${escape(message)}</span>`;
  }

  function loadCard(card) {
    if (card.dataset.loaded === "true" || !historyEnabled) return;
    card.dataset.loaded = "true";
    fetchSeries(card, HOURS_DEFAULT).then(
      (payload) => renderSpark(card, payload),
      (err) => {
        if (!historyEnabled) {
          renderError(card, "History not configured. Start the dashboard with --history-db to enable trends.");
        } else {
          renderError(card, err.message);
        }
      }
    );
  }

  // Set up lazy loading via IntersectionObserver.
  if ("IntersectionObserver" in window) {
    const io = new IntersectionObserver(
      (entries) => {
        entries.forEach((e) => {
          if (e.isIntersecting) {
            loadCard(e.target);
            io.unobserve(e.target);
          }
        });
      },
      { rootMargin: "200px" }
    );
    cards.forEach((c) => io.observe(c));
  } else {
    cards.forEach(loadCard);
  }

  // ---- rendering helpers --------------------------------------------------

  function renderSparkSVG(points) {
    // Two series: CPU request (line) and CPU target (dashed). Keeps the
    // sparkline readable while still showing the relationship Goldilocks
    // is about — current vs recommended.
    const cpuReq = points.map((p) => p.cpuRequest || 0);
    const cpuTgt = points.map((p) => p.cpuTarget || 0);
    const max = Math.max(...cpuReq, ...cpuTgt, 1);
    const svg = svgEl("svg", {
      width: SPARK_WIDTH,
      height: SPARK_HEIGHT,
      viewBox: `0 0 ${SPARK_WIDTH} ${SPARK_HEIGHT}`,
      role: "img",
      "aria-label": "CPU request vs target over the last 24 hours",
    });
    svg.appendChild(linePath(cpuReq, max, SPARK_WIDTH, SPARK_HEIGHT, "var(--color-accent-3-darker)"));
    svg.appendChild(linePath(cpuTgt, max, SPARK_WIDTH, SPARK_HEIGHT, "var(--color-positive)", "3,3"));
    return svg;
  }

  function openExpanded(card) {
    const titleEl = dialog.querySelector(".historyDialog__title");
    const subEl = dialog.querySelector(".historyDialog__sub");
    const chartEl = dialog.querySelector(".historyDialog__chart");
    titleEl.textContent = `${card.dataset.workload} / ${card.dataset.container}`;
    subEl.textContent = `${card.dataset.kind} · ${card.dataset.ns} · last ${HOURS_EXPANDED}h`;
    chartEl.innerHTML = '<p class="historyCard__placeholder">Loading…</p>';
    if (typeof dialog.showModal === "function") {
      dialog.showModal();
    } else {
      dialog.setAttribute("open", "");
    }

    fetchSeries(card, HOURS_EXPANDED).then(
      (payload) => {
        chartEl.innerHTML = "";
        if (!payload.points || payload.points.length === 0) {
          chartEl.innerHTML = '<p class="historyCard__placeholder">No samples yet.</p>';
          return;
        }
        chartEl.appendChild(renderChartSVG(payload.points));
        chartEl.appendChild(renderLegend());
      },
      (err) => {
        chartEl.innerHTML = `<p class="historyCard__placeholder">${escape(err.message)}</p>`;
      }
    );
  }

  function renderChartSVG(points) {
    const cpuReq = points.map((p) => p.cpuRequest || 0);
    const cpuTgt = points.map((p) => p.cpuTarget || 0);
    const cpuLim = points.map((p) => p.cpuLimit || 0);
    const max = Math.max(...cpuReq, ...cpuTgt, ...cpuLim, 1);

    const pad = { top: 16, right: 12, bottom: 28, left: 48 };
    const innerW = CHART_WIDTH - pad.left - pad.right;
    const innerH = CHART_HEIGHT - pad.top - pad.bottom;

    const svg = svgEl("svg", {
      width: "100%",
      height: CHART_HEIGHT,
      viewBox: `0 0 ${CHART_WIDTH} ${CHART_HEIGHT}`,
      role: "img",
      "aria-label": "CPU request, target, and limit over time",
    });

    // gridlines + y-axis labels (4 horizontal lines)
    for (let i = 0; i <= 4; i++) {
      const y = pad.top + (innerH * i) / 4;
      const value = Math.round(max - (max * i) / 4);
      svg.appendChild(svgEl("line", {
        x1: pad.left, x2: CHART_WIDTH - pad.right,
        y1: y, y2: y,
        stroke: "var(--color-border)",
        "stroke-width": 1,
      }));
      const label = svgEl("text", {
        x: pad.left - 6, y: y + 4,
        "text-anchor": "end",
        "font-size": 10,
        fill: "var(--color-text-muted)",
      });
      label.textContent = `${value}m`;
      svg.appendChild(label);
    }

    // x-axis labels (start + end timestamps)
    const fmt = (unix) => {
      const d = new Date(unix * 1000);
      return `${d.getMonth() + 1}/${d.getDate()} ${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
    };
    const xStart = svgEl("text", {
      x: pad.left, y: CHART_HEIGHT - 8,
      "font-size": 10, fill: "var(--color-text-muted)",
    });
    xStart.textContent = fmt(points[0].ts);
    const xEnd = svgEl("text", {
      x: CHART_WIDTH - pad.right, y: CHART_HEIGHT - 8,
      "text-anchor": "end",
      "font-size": 10, fill: "var(--color-text-muted)",
    });
    xEnd.textContent = fmt(points[points.length - 1].ts);
    svg.appendChild(xStart);
    svg.appendChild(xEnd);

    // series — translate the inner-coordinate paths into the padded chart
    const seriesGroup = svgEl("g", { transform: `translate(${pad.left}, ${pad.top})` });
    seriesGroup.appendChild(linePath(cpuLim, max, innerW, innerH, "var(--color-warning)", "5,3"));
    seriesGroup.appendChild(linePath(cpuTgt, max, innerW, innerH, "var(--color-positive)", "3,3"));
    seriesGroup.appendChild(linePath(cpuReq, max, innerW, innerH, "var(--color-accent-3-darker)"));
    svg.appendChild(seriesGroup);

    return svg;
  }

  function renderLegend() {
    const el = document.createElement("div");
    el.className = "historyDialog__legend";
    el.innerHTML = `
      <span><span class="legendSwatch" style="background: var(--color-accent-3-darker)"></span> CPU request</span>
      <span><span class="legendSwatch legendSwatch--dashed" style="background: var(--color-positive)"></span> CPU target</span>
      <span><span class="legendSwatch legendSwatch--dashed" style="background: var(--color-warning)"></span> CPU limit</span>
    `;
    return el;
  }

  function linePath(values, max, w, h, stroke, dash) {
    if (values.length === 0) return svgEl("g");
    const step = values.length > 1 ? w / (values.length - 1) : 0;
    const d = values
      .map((v, i) => {
        const x = i * step;
        const y = h - (v / max) * h;
        return `${i === 0 ? "M" : "L"}${x.toFixed(2)} ${y.toFixed(2)}`;
      })
      .join(" ");
    const path = svgEl("path", {
      d,
      fill: "none",
      stroke,
      "stroke-width": 1.6,
      "stroke-linejoin": "round",
      "stroke-linecap": "round",
    });
    if (dash) path.setAttribute("stroke-dasharray", dash);
    return path;
  }

  function svgEl(name, attrs) {
    const el = document.createElementNS("http://www.w3.org/2000/svg", name);
    if (attrs) {
      for (const [k, v] of Object.entries(attrs)) el.setAttribute(k, v);
    }
    return el;
  }

  function escape(s) {
    return String(s).replace(/[&<>"']/g, (c) => ({
      "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
    }[c]));
  }

  function ensureDialog() {
    let d = document.getElementById("js-history-dialog");
    if (d) return d;
    d = document.createElement("dialog");
    d.id = "js-history-dialog";
    d.className = "historyDialog";
    d.innerHTML = `
      <header class="historyDialog__header">
        <div>
          <h3 class="historyDialog__title">Trend</h3>
          <p class="historyDialog__sub"></p>
        </div>
        <button type="button" class="historyDialog__close linkPill linkPill--ghost" aria-label="Close">
          <i aria-hidden="true" class="fas fa-fw fa-times"></i>
        </button>
      </header>
      <div class="historyDialog__chart"></div>
    `;
    document.body.appendChild(d);
    d.querySelector(".historyDialog__close").addEventListener("click", () => d.close());
    d.addEventListener("click", (e) => {
      if (e.target === d) d.close(); // click on backdrop
    });
    return d;
  }
})();
