// Bulk export of Goldilocks recommendations as YAML, JSON, or CSV.
// Two scopes:
//   * Per-namespace — buttons inside each namespace card (.js-export-ns)
//   * All namespaces — a single top-of-page button (.js-export-all)
// Reads data from the rendered DOM so client-side filters (state filter,
// search filter, scaled-to-zero hiding) compose naturally: only currently
// visible workloads are exported.
(function () {
  const CSV_COLUMNS = [
    "namespace",
    "workload_type",
    "workload_name",
    "container",
    "state",
    "cpu_request_current",
    "cpu_limit_current",
    "memory_request_current",
    "memory_limit_current",
    "cpu_target",
    "memory_target",
    "cpu_lower_bound",
    "cpu_upper_bound",
    "memory_lower_bound",
    "memory_upper_bound",
  ];

  function cssEscape(value) {
    if (window.CSS && CSS.escape) return CSS.escape(value);
    return String(value).replace(/[^a-zA-Z0-9_-]/g, "\\$&");
  }

  function namespaceArticles(scopeNamespace) {
    if (scopeNamespace) {
      const a = document.querySelector(
        `article[data-namespace="${cssEscape(scopeNamespace)}"]`
      );
      return a ? [a] : [];
    }
    return Array.from(
      document.querySelectorAll("article[data-namespace]:not([hidden])")
    );
  }

  function gatherEntries(scopeNamespace) {
    const entries = [];
    namespaceArticles(scopeNamespace).forEach((article) => {
      const namespace = article.dataset.namespace;
      article.querySelectorAll(".js-workload:not([hidden])").forEach((wl) => {
        const wType = wl.dataset.workloadType || "";
        const wName = wl.dataset.workloadName || "";

        wl.querySelectorAll(".detailInfo.--container:not([hidden])").forEach(
          (cc) => {
            entries.push({
              namespace,
              workload: { type: wType, name: wName },
              container: cc.dataset.containerName || "",
              state: cc.dataset.state || "",
              current: {
                cpuRequest: cc.dataset.cpuRequest || "",
                cpuLimit: cc.dataset.cpuLimit || "",
                memoryRequest: cc.dataset.memRequest || "",
                memoryLimit: cc.dataset.memLimit || "",
              },
              target: {
                cpu: cc.dataset.cpuTarget || "",
                memory: cc.dataset.memTarget || "",
              },
              bounds: {
                cpuLower: cc.dataset.cpuLower || "",
                cpuUpper: cc.dataset.cpuUpper || "",
                memoryLower: cc.dataset.memLower || "",
                memoryUpper: cc.dataset.memUpper || "",
              },
              recommendationYaml: yamlFor(cc),
            });
          }
        );
      });
    });
    return entries;
  }

  function yamlFor(containerEl) {
    const code = containerEl.querySelector("code.language-yaml");
    return code ? code.textContent.trim() : "";
  }

  function csvEscape(value) {
    const s = String(value ?? "");
    if (/[",\n\r]/.test(s)) {
      return `"${s.replace(/"/g, '""')}"`;
    }
    return s;
  }

  function buildCsv(entries) {
    const rows = [CSV_COLUMNS.join(",")];
    entries.forEach((e) => {
      rows.push(
        [
          e.namespace,
          e.workload.type,
          e.workload.name,
          e.container,
          e.state,
          e.current.cpuRequest,
          e.current.cpuLimit,
          e.current.memoryRequest,
          e.current.memoryLimit,
          e.target.cpu,
          e.target.memory,
          e.bounds.cpuLower,
          e.bounds.cpuUpper,
          e.bounds.memoryLower,
          e.bounds.memoryUpper,
        ]
          .map(csvEscape)
          .join(",")
      );
    });
    return rows.join("\n") + "\n";
  }

  function buildYaml(entries) {
    if (!entries.length) return "# No recommendations found\n";
    return entries
      .map((e) => {
        const header = `# ${e.workload.type}/${e.workload.name} (namespace: ${e.namespace}, container: ${e.container})`;
        return `${header}\n${e.recommendationYaml}\n`;
      })
      .join("\n---\n");
  }

  function buildJson(entries) {
    return JSON.stringify(entries, null, 2);
  }

  function downloadBlob(filename, contentType, body) {
    const blob = new Blob([body], { type: contentType });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    setTimeout(() => URL.revokeObjectURL(url), 0);
  }

  function exportScoped(scopeNamespace, format, filenameStem) {
    const entries = gatherEntries(scopeNamespace);
    switch (format) {
      case "csv":
        downloadBlob(`${filenameStem}.csv`, "text/csv;charset=utf-8", buildCsv(entries));
        break;
      case "json":
        downloadBlob(`${filenameStem}.json`, "application/json", buildJson(entries));
        break;
      default:
        downloadBlob(`${filenameStem}.yaml`, "text/yaml", buildYaml(entries));
    }
  }

  document.addEventListener("click", function (event) {
    const nsButton = event.target.closest(".js-export-ns");
    if (nsButton) {
      event.preventDefault();
      const namespace = nsButton.dataset.namespace;
      const format = (nsButton.dataset.format || "yaml").toLowerCase();
      exportScoped(namespace, format, `goldilocks-${namespace}`);
      return;
    }
    const allButton = event.target.closest(".js-export-all");
    if (allButton) {
      event.preventDefault();
      const format = (allButton.dataset.format || "csv").toLowerCase();
      exportScoped(null, format, "goldilocks-all-namespaces");
    }
  });
})();
