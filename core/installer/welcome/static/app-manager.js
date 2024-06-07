document.addEventListener("DOMContentLoaded", function () {
    const page = document.documentElement;
    const headerHeight = parseFloat(getComputedStyle(page).getPropertyValue('--pico-header-height').replace("px", ""));
    const nav = document.getElementById("menu");
    const windowHeight = window.innerHeight - headerHeight;
    nav.style.setProperty("--max-height", `${windowHeight}px`);
});

let prevWindowHeight = window.innerHeight;

window.addEventListener("resize", function () {
    const nav = document.getElementById("menu");
    const windowHeight = window.innerHeight;
    const heightDiff = prevWindowHeight - windowHeight;
    const currentMaxHeight = parseFloat(nav.style.getPropertyValue("--max-height").replace("px", ""));
    if (!isNaN(currentMaxHeight)) {
        const newMaxHeight = currentMaxHeight - heightDiff;
        nav.style.setProperty("--max-height", `${newMaxHeight}px`);
    }
    prevWindowHeight = windowHeight;
});
