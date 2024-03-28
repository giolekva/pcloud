const confirmationDialog = document.getElementById("confirmation");
const cancelButton = document.getElementById("cancel-button");
const confirmButton = document.getElementById("confirm-button");

let formToRemove;

function showConfirmationDialog(form) {
    formToRemove = form;
    const message = formToRemove.dataset.confirmationMessage;
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
                ff = await handleRemoveOwnerSubmit(form);
                if (ff) {
                    form.submit();
                }
            } catch (error) {
                console.error(error);
            }
        });
    });
});
