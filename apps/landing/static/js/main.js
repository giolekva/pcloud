let mouseOverEnabled = true;

function handleImageChange(imageURL, index, factTitle) {
    const factImage = document.getElementById("factImage");
    if (factImage.src.endsWith(imageURL)) {
        return;
    };
    highlightFact(index);
    factImage.classList.add("fade-out");
    setTimeout(() => {
        factImage.src = imageURL;
        factImage.alt = factTitle;
        factImage.classList.remove("fade-out");
    }, 300);
};

function highlightFact(index) {
    const facts = document.querySelectorAll(".facts");
    facts.forEach((fact, i) => {
        if (i === index) {
            fact.classList.add("active-fact");
        } else {
            fact.classList.remove("active-fact");
        };
    });
};

function handleMouseover(imageURL, index, factTitle) {
    if (!mouseOverEnabled) {
        return;
    };
    handleImageChange(imageURL, index, factTitle);
};

let scrollTimeout;
let mouseoverTimeout;

function delayScroll(func, delay) {
    return function (...args) {
        clearTimeout(scrollTimeout);
        scrollTimeout = setTimeout(() => {
            func.apply(this, args);
        }, delay);
    };
}

function delayMouseoverEnable(delay) {
    clearTimeout(mouseoverTimeout);
    mouseoverTimeout = setTimeout(() => {
        mouseOverEnabled = true;
    }, delay);
}

let lastScrollTop = 0;

function getFirstVisibleFact() {
    const facts = document.querySelectorAll(".facts");
    const factImage = document.getElementById("factImageBox");
    const imageBottom = factImage.getBoundingClientRect().bottom;

    const scrollTop = document.documentElement.scrollTop || document.body.scrollTop;
    const isScrollingDown = scrollTop > lastScrollTop;
    lastScrollTop = scrollTop;

    if (isScrollingDown) {
        for (let i = 0; i < facts.length; i++) {
            const rect = facts[i].getBoundingClientRect();
            if (rect.top < window.innerHeight && rect.top >= imageBottom) {
                return i;
            };
        };
        return 7;
    } else {
        for (let i = 0; i < facts.length; i++) {
            const rect = facts[i].getBoundingClientRect();
            if (rect.top >= 0 && rect.top >= imageBottom) {
                return i;
            };
        };
    };
    return 0;
};

document.addEventListener("DOMContentLoaded", function () {
    const gridContainer = document.querySelector(".grid-container");
    const facts = JSON.parse(gridContainer.dataset.facts);

    const handleScroll = () => {
        clearTimeout(scrollTimeout);
        mouseOverEnabled = false;
        scrollTimeout = setTimeout(() => {
            const currentSection = getFirstVisibleFact();
            handleImageChange(facts[currentSection].params.image, currentSection, facts[currentSection].params.title);
            delayMouseoverEnable(800);
        }, 300);
    };

    const checkAddRemoveScrollListener = () => {
        if (window.innerWidth <= 768) {
            window.addEventListener("scroll", handleScroll);
        } else {
            window.removeEventListener("scroll", handleScroll);
        }
    };

    checkAddRemoveScrollListener();

    let resizing = false;
    const resizeDelayTime = 1000;
    let resizeTimeout;

    const startResizing = () => {
        clearTimeout(resizeTimeout);
        resizing = true;
        document.querySelectorAll(".facts").forEach(fact => {
            fact.removeEventListener("mouseover", handleMouseover);
        });
    };

    const stopResizing = () => {
        clearTimeout(resizeTimeout);
        resizeTimeout = setTimeout(() => {
            resizing = false;
            checkAddRemoveScrollListener();
            document.querySelectorAll(".facts").forEach((fact, index) => {
                fact.addEventListener("mouseover", () => handleMouseover(facts[index].params.image, index, facts[index].params.title));
            });
        }, resizeDelayTime);
    };

    window.addEventListener("resize", () => {
        if (!resizing) {
            startResizing();
        }
        stopResizing();
    });

    document.querySelectorAll(".facts").forEach((fact, index) => {
        fact.addEventListener("mouseover", () => handleMouseover(facts[index].params.image, index, facts[index].params.title));
    });
});

async function loadPublicData() {
  let networkSelect = document.querySelector("select#network");
  if (networkSelect === undefined) {
	return;
  }
  networkSelect.innerHTML = "";
  let appTypeSelect = document.querySelector("select#app-type");
  if (appTypeSelect === undefined) {
	return;
  }
  appTypeSelect.innerHTML = "";
  let resp = await fetch("https://app.v1.dodo.cloud/api/public-data");
  if (!resp.ok) {
	return;
  }
  let data = await resp.json();
  data.networks.forEach((network) => {
	let opt = document.createElement("option");
	opt.setAttribute("value", network.domain);
	opt.innerHTML = network.domain;
	networkSelect.appendChild(opt);
  });
  data.types.forEach((t) => {
	let opt = document.createElement("option");
	opt.setAttribute("value", t);
	opt.innerHTML = t;
	appTypeSelect.appendChild(opt);
  });
}

function register() {
  var data = {
	type: document.getElementById("app-type").value,
	adminPublicKey: document.getElementById("public-key").value,
	network: document.getElementById("network").value,
	subdomain: document.getElementById("subdomain").value,
  };
  fetch("https://app.v1.dodo.cloud/api/apps", {
	method: "POST",
	body: JSON.stringify(data),
  }).then((resp) => {
	resp.json().then((r) => {
	  console.log(r);
	});
  });
  return false;
}

document.addEventListener("DOMContentLoaded", () => {
  loadPublicData();
});
