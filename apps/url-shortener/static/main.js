function toggle(name, status) {
  const data = {
    name: name,
    active: status,
  };
  fetch("/api/update/", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(data),
  })
    .then((response) => {
      if (response.ok) {
        window.location.reload();
      }
    })
    .catch((error) => {
      console.error("Error:", error);
    });
}
