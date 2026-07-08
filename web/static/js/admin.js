// Progressive enhancement only: every form/select here already works
// without JS (a visible submit button is always present). This just adds
// a confirmation prompt before destructive submits, and auto-submits
// filter dropdowns so the visible button becomes a no-JS fallback.
document.addEventListener("submit", (event) => {
  const form = event.target;
  const message = form.dataset.confirm;
  if (message && !window.confirm(message)) {
    event.preventDefault();
  }
});

document.addEventListener("change", (event) => {
  if (event.target.matches("[data-autosubmit]")) {
    event.target.form.requestSubmit();
  }
});
