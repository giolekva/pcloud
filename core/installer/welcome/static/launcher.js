document.addEventListener("DOMContentLoaded", function () {
  document.getElementById('appFrame-default').contentDocument.write("Welcome to the dodo: application launcher, think of it as your desktop environment. You can launch applications from left-hand side dock. You should setup VPN clients on your devices, so you can install applications from Application Manager and access your private network. Instructions on how to do that can be viewed by clicking <b>Help</b> button after hovering over <b>Headscale</b> icon in the dock.");

  function showTooltip(obj) {
    obj.style.visibility = 'visible';
    obj.style.opacity = '1';
  }
  function hideTooltip(obj) {
    obj.style.visibility = 'hidden';
    obj.style.opacity = '0';
  }

  const circle = document.querySelector(".user-circle");
  const tooltipUser = document.querySelector("#tooltip-user");
  [
    ['mouseenter', () => showTooltip(tooltipUser)],
    ['mouseleave', () => hideTooltip(tooltipUser)],
  ].forEach(([event, listener]) => {
    circle.addEventListener(event, listener);
  });

  const iframes = {};
  const rightPanel = document.getElementById('right-panel');

  function showIframe(appId) {
    document.querySelectorAll('.appFrame').forEach(iframe => {
      iframe.style.display = iframe.id === `appFrame-${appId}` ? 'block' : 'none';
    });
  }

  function createIframe(appId, appUrl) {
    const iframe = document.createElement('iframe');
    iframe.id = `appFrame-${appId}`;
    iframe.className = 'appFrame';
    iframe.src = appUrl;
    iframe.style.display = 'none';
    rightPanel.appendChild(iframe);
    iframes[appId] = iframe;
  }

  const icons = document.querySelectorAll(".app-icon-tooltip");
  icons.forEach(function (icon) {
    icon.addEventListener("click", function (event) {
      event.stopPropagation();
      const appUrl = this.getAttribute("data-app-url");
      const appId = this.getAttribute("data-app-id");
      if (!appUrl) {
        const modalId = `modal-${this.querySelector('.help-button').id.replace('help-button-', '')}`;
        openModal(document.getElementById(modalId));
      } else {
        if (!iframes[appId]) createIframe(appId, appUrl);
        showIframe(appId);
      }
      document.querySelectorAll(".app-icon-tooltip .background-glow").forEach((e) => e.remove());
      const glow = document.createElement('div');
      glow.classList.add("background-glow");
      glow.setAttribute("style", "transform: none; transform-origin: 50% 50% 0px;")
      this.appendChild(glow);
    });
    const tooltip = icon.querySelector('.tooltip');
    tooltip.addEventListener("click", function (event) {
      event.stopPropagation();
    });
    [
      ['mouseenter', () => showTooltip(tooltip)],
      ['mouseleave', () => hideTooltip(tooltip)],
      ['focus', () => showTooltip(tooltip)],
      ['blur', () => hideTooltip(tooltip)],
    ].forEach(([event, listener]) => {
      icon.addEventListener(event, listener);
    });
  });

  let visibleModal = undefined;
  const openModal = function (modal) {
    modal.removeAttribute("close");
    modal.setAttribute("open", true);
    visibleModal = modal;
  };
  const closeModal = function (modal) {
    modal.removeAttribute("open");
    modal.setAttribute("close", true);
    visibleModal = undefined;
  };

  const helpButtons = document.querySelectorAll('.help-button');
  helpButtons.forEach(function (button) {
    button.addEventListener('click', function (event) {
      event.stopPropagation();
      const buttonId = button.getAttribute('id');
      const modalId = 'modal-' + buttonId.substring("help-button-".length);
      const closeHelpId = "close-help-" + buttonId.substring("help-button-".length);
      const modal = document.getElementById(modalId);
      openModal(modal);
      const closeHelpButton = document.getElementById(closeHelpId);
      closeHelpButton.addEventListener('click', function (event) {
        event.stopPropagation();
        closeModal(modal);
      });
    });
  });

  const modalHelpButtons = document.querySelectorAll('.title-menu');
  modalHelpButtons.forEach(function (button) {
    button.addEventListener('click', function (event) {
      event.stopPropagation();
      const helpTitle = button.getAttribute('id');
      const helpTitleId = helpTitle.substring('title-'.length);
      const helpContentId = 'help-content-' + helpTitleId;
      const allContentElements = document.querySelectorAll('.help-content');
      allContentElements.forEach(function (contentElement) {
        contentElement.style.display = "none";
      });
      modalHelpButtons.forEach(function (button) {
        button.removeAttribute("aria-current");
      });
      document.getElementById(helpContentId).style.display = 'block';
      button.setAttribute("aria-current", "page");
    });
  });

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape" && visibleModal) {
      closeModal(visibleModal);
    }
  });

  document.addEventListener("click", (event) => {
    if (visibleModal === null) return;
    const modalContent = visibleModal.querySelector("article");
    const closeButton = visibleModal.querySelector(".close-button");
    if (!modalContent.contains(event.target) || closeButton.contains(event.target)) {
      closeModal(visibleModal);
    }
  });
});

function copyToClipboard(elem, text) {
  navigator.clipboard.writeText(text);
  elem.setAttribute("data-tooltip", "Copied");
  elem.setAttribute("data-placement", "bottom");
  setTimeout(() => {
	elem.removeAttribute("data-tooltip");
	elem.removeAttribute("data-placement");
  }, 500);
}
