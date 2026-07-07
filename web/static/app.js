document.addEventListener("submit", function (event) {
    const form = event.target;
    const message = form.getAttribute("data-confirm");
    if (message && !window.confirm(message)) {
        event.preventDefault();
    }
});

document.addEventListener("click", function (event) {
    const addButton = event.target.closest("[data-add-line-item]");
    if (addButton) {
        const container = document.querySelector("[data-line-items]");
        const first = container?.querySelector("[data-line-item]");
        if (!container || !first) return;
        const clone = first.cloneNode(true);
        clone.querySelectorAll("input").forEach((input) => {
            if (input.name === "item_quantity") input.value = "1";
            else if (input.name === "item_tax_rate") input.value = "19";
            else input.value = "";
        });
        container.appendChild(clone);
        return;
    }

    const removeButton = event.target.closest("[data-remove-line-item]");
    if (removeButton) {
        const container = document.querySelector("[data-line-items]");
        const rows = container?.querySelectorAll("[data-line-item]") || [];
        if (rows.length <= 1) {
            rows[0]?.querySelectorAll("input").forEach((input) => {
                if (input.name === "item_quantity") input.value = "1";
                else if (input.name === "item_tax_rate") input.value = "19";
                else input.value = "";
            });
            return;
        }
        removeButton.closest("[data-line-item]")?.remove();
    }
});
