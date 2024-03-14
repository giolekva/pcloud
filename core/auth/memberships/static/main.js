document.addEventListener("DOMContentLoaded", function () {
  document.querySelectorAll(".group-link").forEach((link) => {
    link.addEventListener("click", function (event) {
      event.preventDefault();
      const groupName = event.target.textContent;
      fetch(`/group/${groupName}`)
        .then((response) => {
          if (!response.ok) {
            throw new Error("Failed to fetch group data");
          }
          return response.text();
        })
        .then((htmlContent) => {
          document.body.innerHTML = htmlContent;
        })
        .catch((error) => console.error(error));
    });
  });
});
