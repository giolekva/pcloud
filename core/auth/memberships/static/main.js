const confirmationDialog = document.getElementById("confirmation");
const cancelButton = document.getElementById("cancel-button");
const confirmButton = document.getElementById("confirm-button");

const errorCancelButton = document.getElementById("error-cancel-button");
const errorMessageDialog = document.getElementById("error-message");

let activeModalStatus = false;
let activeModal = '';

function keydownHandler(event) {
    if (event.key === "Escape" && activeModalStatus && activeModal === "confirmation") {
        hideConfirmationDialog();
    }
}

function oustideModalHandler(event) {
    if (activeModalStatus && confirmationDialog === event.target) {
        hideConfirmationDialog();
        errorMessageDialog.close();
    }
}

function showConfirmationDialog(form) {
    activeModalStatus = true;
    activeModal = "confirmation";
    document.addEventListener("keydown", keydownHandler);
    document.addEventListener("click", oustideModalHandler);
    const message = form.dataset.confirmationMessage;
    document.getElementById("confirmation-message").innerHTML = message;
    confirmationDialog.showModal();
    let confirmed;
    let p = new Promise((resolve) => {
        confirmed = resolve;
    });
    confirmButton.onclick = () => {
        confirmed(true);
    };
    cancelButton.onclick = () => {
        hideConfirmationDialog();
        confirmed(false);
    };
    return p;
}

function hideConfirmationDialog() {
    activeModalStatus = false;
    activeModal = '';
    document.removeEventListener("keydown", keydownHandler);
    document.removeEventListener("click", oustideModalHandler);
    confirmationDialog.close();
}

async function handleRemoveOwnerSubmit(form) {
    event.preventDefault();
    return await showConfirmationDialog(form);
}

document.addEventListener("DOMContentLoaded", function () {
    const removeOwnerForms = document.querySelectorAll(".remove-form");
    removeOwnerForms.forEach((form) => {
        form.addEventListener("submit", async function (event) {
            event.preventDefault();
            try {
                isConfirmed = await handleRemoveOwnerSubmit(form);
                if (isConfirmed) {
                    form.submit();
                }
            } catch (error) {
                console.error(error);
            }
        });
    });

    errorCancelButton.addEventListener("click", function () {
        errorMessageDialog.close();
    });

    document.addEventListener("keydown", function (event) {
        if (event.key === "Escape") {
            errorMessageDialog.close();
        }
    });
    document.addEventListener("click", function (event) {
        if (errorMessageDialog === event.target) {
            errorMessageDialog.close();
        }
    });
});
