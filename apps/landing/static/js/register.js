async function loadPublicData() {
    let networkSelect = document.querySelector("select#network");
    if (networkSelect === undefined) {
        return;
    }
    let appTypeSelect = document.querySelector("select#app-type");
    if (appTypeSelect === undefined) {
        return;
    }
    networkSelect.innerHTML = `<option value="" disabled selected>domain</option>`;
    appTypeSelect.innerHTML = `<option value="" disabled selected>application type</option>`;
    let resp = await fetch(`${apiBaseURL}/api/public-data`);
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

function errorRender(error) {
    const errorMsg = document.getElementById("error-message");
    errorMsg.innerHTML = error;
    errorMsg.style.display = "block";
}

function triggerForm(status, errDisplay, buttonTxt, spinnerStatus) {
    const form = document.getElementById('register-form');
    const elements = form.querySelectorAll('input, select, textarea, button');
    elements.forEach(element => {
        element.disabled = status;
    });
    const errorMsg = document.getElementById("error-message");
    errorMsg.style.display = errDisplay;
    const button = document.getElementById("create-app-button");
    button.removeChild(button.lastChild);
    button.appendChild(document.createTextNode(buttonTxt));
    const spinner = document.getElementById("spinner");
    spinner.style.display = spinnerStatus;
}

async function register(event) {
    event.preventDefault();
    const data = {
        type: document.getElementById("app-type").value,
        adminPublicKey: document.getElementById("public-key").value,
        network: document.getElementById("network").value,
        subdomain: document.getElementById("subdomain").value,
    };
    triggerForm(true, "none", "\u00A0\u00A0\creating first app", "inline-block");
    fetch(`${apiBaseURL}/api/apps`, {
        method: "POST",
        body: JSON.stringify(data)
    })
        .then(response => {
            if (!response.ok) {
                errorRender("Internal error, try again");
                triggerForm(false, "block", "create first app", "none");
            }
            return response.json();
        })
        .then(result => {
            const domain = document.getElementById("network").value;
            const subdomain = document.getElementById("subdomain").value;
            const appLink = `https://${subdomain}.${domain}`;
            const appStatusLink = `https://status.${subdomain}.${domain}`;

            const successHTML = `
            <div class="registration-outcome">
                <h3>Application has been successfully deployed, use information below to access it:</h3>
                <button onclick="window.open('${appLink}', '_blank')">Application address: ${appLink}</button>
                <br>
                <button onclick="window.open('${appStatusLink}', '_blank')">Status page address: ${appStatusLink}</button>
                <br>
                <button class="pass" onclick="copyPassword('${result.password}')" id="copy-button">
                    Status page password:&nbsp;<strong>${result.password}</strong> 
                    <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 256 256">
                        <path fill="currentColor" d="M216 32H88a8 8 0 0 0-8 8v40H40a8 8 0 0 0-8 8v128a8 8 0 0 0 8 8h128a8 8 0 0 0 8-8v-40h40a8 8 0 0 0 8-8V40a8 8 0 0 0-8-8m-56 176H48V96h112Zm48-48h-32V88a8 8 0 0 0-8-8H96V48h112Z" />
                    </svg>
                </button>
                <div id="tooltip" class="tooltip">Password copied to clipboard</div>
            </div>`;
            document.getElementById("form-container").innerHTML = successHTML;
        })
        .catch(error => {
            errorRender(`Failed to deploy application. Error: '${error.message}'`);
            triggerForm(false, "block", "create first app", "none");
        })
        .finally(() => {
            document.getElementById("spinner").style.display = "none";
        });
    return;
}

function copyPassword(password) {
    navigator.clipboard.writeText(password).then(() => {
        const button = document.getElementById("copy-button");
        const tooltip = document.getElementById("tooltip");
        const rect = button.getBoundingClientRect();
        const tooltipWidth = tooltip.offsetWidth;
        tooltip.style.top = `${rect.top - 30 + window.scrollY}px`;
        tooltip.style.left = `${rect.left + (rect.width / 2) - (tooltipWidth / 2) + window.scrollX}px`;
        tooltip.style.opacity = "1";
        tooltip.style.visibility = "visible";
        setTimeout(() => {
            tooltip.style.opacity = "0";
            tooltip.style.visibility = "hidden";
        }, 1000);
    });
}

document.addEventListener("DOMContentLoaded", () => {
    loadPublicData();
    const registerForm = document.getElementById("register-form");
    if (registerForm) {
        registerForm.onsubmit = register;
    }
});
