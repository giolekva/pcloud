function triggerForm(status, buttonTxt) {
    const form = document.getElementById("create-app");
    const elements = form.querySelectorAll("input, select, textarea, button");
    const button = document.getElementById("create-app-button");
    button.textContent = buttonTxt;
    button.setAttribute("aria-busy", status);
    elements.forEach(element => {
        element.disabled = status;
    });
}

document.addEventListener("DOMContentLoaded", () => {
    const form = document.getElementById("create-app");
    form.addEventListener("submit", (event) => {
        setTimeout(() => {
            triggerForm(true, "creating app ...");
        }, 0);
    });
    triggerForm(false, "create app");
});
