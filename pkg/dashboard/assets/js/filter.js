import {
    showElement,
    hideElement
} from "./utilities.js";

const form = document.getElementById("js-filter-form");
const container = document.getElementById("js-filter-container");

const filterInput = form?.querySelector("input[type='text']");
// On the dashboard page namespaces are <article data-filter>; on the
// namespace list page they're <li data-filter>. Workload rows also have
// data-filter but are tagged with .js-workload so we can exclude them here.
const namespaces = container?.querySelectorAll("[data-filter]:not(.js-workload)");
const workloads = container?.querySelectorAll(".js-workload[data-filter]");

const outputVisual = form?.querySelector("output[aria-hidden]");
const outputPolite = form?.querySelector("output[aria-live='polite']");
const outputAlert = form?.querySelector("output[role='alert']");

let statusDelay = null;

if (!form) {
    console.error("Could not find filter form");
} else if (!filterInput) {
    hideElement(form);
    console.error("Could not find filter input element, removed filter form");
} else if (!container) {
    hideElement(form);
    console.error("Could not find filter results container, removed filter form");
} else if (!outputVisual || !outputPolite || !outputAlert) {
    hideElement(form);
    console.error("Could not find all filter output elements, removed filter form");
} else if (!namespaces || namespaces.length === 0) {
    hideElement(form);
    console.error("No filterable entries found, removed filter form");
} else {
    filterInput.addEventListener("input", runFilter);
    runFilter();
}

function runFilter() {
    updateResults();
    updateStatus();
}

function updateResults() {
    const filterTerm = filterInput.value;

    if (!filterTerm) {
        clearFilter();
        return;
    }

    const regex = new RegExp(`${filterTerm.trim().replace(/\s+/g, "|")}`, "i");

    // First pass: mark each workload match/no-match
    workloads?.forEach((wl) => {
        if (regex.test(wl.dataset.filter)) {
            showElement(wl);
        } else {
            hideElement(wl);
        }
    });

    // Second pass: a namespace is visible if it matches itself OR has any visible workloads
    namespaces.forEach((ns) => {
        const nsMatches = regex.test(ns.dataset.filter);
        const hasVisibleWorkload = ns.querySelector(".js-workload:not([hidden])") !== null;
        if (nsMatches || hasVisibleWorkload) {
            showElement(ns);
            if (nsMatches && !hasVisibleWorkload) {
                ns.querySelectorAll(".js-workload").forEach(showElement);
            }
        } else {
            hideElement(ns);
        }
    });
}

function clearFilter() {
    namespaces.forEach(showElement);
    workloads?.forEach(showElement);
}

function updateStatus() {
    const visibleNamespaces = container?.querySelectorAll("[data-filter]:not(.js-workload):not([hidden])").length || 0;
    const visibleWorkloads = container?.querySelectorAll(".js-workload:not([hidden])").length || 0;
    const totalNamespaces = namespaces.length;
    const hasWorkloads = (workloads?.length || 0) > 0;

    let message, type;

    if (!filterInput.value) {
        message = `${totalNamespaces} namespaces found`;
        type = "polite";
    } else if (visibleNamespaces === 0) {
        message = "No matches";
        type = "alert";
    } else if (hasWorkloads) {
        message = `${visibleNamespaces} of ${totalNamespaces} namespaces · ${visibleWorkloads} workloads`;
        type = "polite";
    } else {
        message = `${visibleNamespaces} of ${totalNamespaces} namespaces`;
        type = "polite";
    }

    changeStatusMessage(message, type);
}

function changeStatusMessage(message, type = "polite") {
    if (statusDelay) {
        window.clearTimeout(statusDelay);
    }

    outputVisual.textContent = message;
    outputPolite.textContent = "";
    outputAlert.textContent = "";

    statusDelay = window.setTimeout(() => {
        switch (type) {
            case "polite":
                outputPolite.textContent = message;
                outputAlert.textContent = "";
                break;
            case "alert":
                outputPolite.textContent = "";
                outputAlert.textContent = message;
                break;
            default:
                outputPolite.textContent = "Error: There was a problem with the filter.";
                outputAlert.textContent = "";
        }
    }, 1000);
}
