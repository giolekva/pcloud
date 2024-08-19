function delaySearch(func, wait) {
    let timeout;
    return function (...args) {
        clearTimeout(timeout);
        timeout = setTimeout(() => func.apply(this, args), wait);
    };
}

document.addEventListener("DOMContentLoaded", function () {
    let searchRequestCount = 0;
    const page = document.documentElement;
    const headerHeight = parseFloat(getComputedStyle(page).getPropertyValue("--pico-header-height").replace("px", ""));
    const nav = document.getElementById("menu");
    const windowHeight = window.innerHeight - headerHeight;
    nav.style.setProperty("--max-height", `${windowHeight}px`);
    const menu = document.getElementById("menu-nav");
    const menuHeight = parseFloat(getComputedStyle(document.getElementById('menu-nav')).height.replace("px", "")) + 15;
    menu.style.setProperty("height", `${menuHeight}px`);
    const searchForm = document.getElementById("search-form");
    const searchInput = document.getElementById("search-input");
    function startSearchAnimation() {
        searchInput.classList.remove("search-icon");
        searchInput.classList.add("search-progress-icon");
    }
    function stopSearchAnimation() {
        searchInput.classList.remove("search-progress-icon");
        searchInput.classList.add("search-icon");
    }
    function fetchAndUpdateAppList() {
        searchRequestCount++;
        const currentRequest = searchRequestCount;
        const formData = new FormData(searchForm);
        const query = formData.get("query");
        const pageType = document.getElementById("page-type").value;
        const url = `/${pageType}?query=${encodeURIComponent(query)}`;
        startSearchAnimation();
        fetch(url, {
            method: "GET"
        })
            .then(response => response.text())
            .then(html => {
                if (currentRequest !== searchRequestCount) {
                    return;
                }
                const tempDiv = document.createElement("div");
                tempDiv.innerHTML = html;
                const newAppListHTML = tempDiv.querySelector("#app-list").innerHTML;
                const appListContainer = document.getElementById("app-list");
                appListContainer.innerHTML = newAppListHTML;
            })
            .catch(error => {
                console.error("Error fetching app list:", error);
            })
            .finally(() => {
                stopSearchAnimation();
            });
    }
    const delayedFetchAndUpdateAppList = delaySearch(fetchAndUpdateAppList, 300);
    if (searchForm) {
        searchForm.addEventListener("submit", (event) => {
            event.preventDefault();
            fetchAndUpdateAppList();
        });
    }
    if (searchInput) {
        searchInput.addEventListener("input", () => {
            delayedFetchAndUpdateAppList();
        });
    }
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
