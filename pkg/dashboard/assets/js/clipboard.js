// Wires up "Copy" buttons that target a sibling code block via data-copy-target.
(function () {
  function flash(button, message) {
    const original = button.innerHTML;
    button.innerHTML = `<i aria-hidden="true" class="fas fa-fw fa-check"></i> ${message}`;
    button.classList.add("is-copied");
    setTimeout(() => {
      button.innerHTML = original;
      button.classList.remove("is-copied");
    }, 1600);
  }

  function copyText(text) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      return navigator.clipboard.writeText(text);
    }
    return new Promise((resolve, reject) => {
      const ta = document.createElement("textarea");
      ta.value = text;
      ta.setAttribute("readonly", "");
      ta.style.position = "absolute";
      ta.style.left = "-9999px";
      document.body.appendChild(ta);
      ta.select();
      try {
        document.execCommand("copy");
        resolve();
      } catch (err) {
        reject(err);
      } finally {
        document.body.removeChild(ta);
      }
    });
  }

  document.addEventListener("click", function (event) {
    const button = event.target.closest(".js-copy");
    if (!button) return;
    event.preventDefault();

    const targetSel = button.getAttribute("data-copy-target");
    const target = targetSel ? document.querySelector(targetSel) : null;
    if (!target) {
      console.warn("Copy target not found:", targetSel);
      return;
    }

    copyText(target.textContent).then(
      () => flash(button, "Copied"),
      () => flash(button, "Failed")
    );
  });
})();
