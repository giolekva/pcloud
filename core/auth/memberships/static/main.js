const confirmationDialog = document.getElementById("confirmation");
const cancelButton = document.getElementById("cancel-button");
const confirmButton = document.getElementById("confirm-button");

function showConfirmationDialog(form) {
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
});

document.addEventListener("DOMContentLoaded", function () {
    var errorCancelButton = document.getElementById("error-cancel-button");
    var errorMessageDialog = document.getElementById("error-message");
    errorCancelButton.addEventListener("click", function () {
        errorMessageDialog.close();
    });
});
