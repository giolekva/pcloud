function showTooltip(obj) {
  obj.style.visibility = 'visible';
  obj.style.opacity = '1';
}
function hideTooltip(obj) {
  obj.style.visibility = 'hidden';
  obj.style.opacity = '0';
}

document.addEventListener("DOMContentLoaded", function () {
  document.getElementById('appFrame-default').contentDocument.write("Welcome to the dodo: application launcher, think of it as your desktop environment. You can launch applications from left-hand side dock. You should setup VPN clients on your devices, so you can install applications from Application Manager and access your private network. Instructions on how to do that can be viewed by clicking <b>Help</b> button after hovering over <b>Headscale</b> icon in the dock.");
  document.getElementById('appFrame-default').style.backgroundColor = '#d6d6d6';
  const icons = document.querySelectorAll(".app-icon");
  const circle = document.querySelector(".user-circle");
  const tooltipUser = document.querySelector("#tooltip-user");
  const initial = document.getElementById('user-initial');

  circle.addEventListener('mouseenter', () => {
    icons.forEach(icon => {
      const tooltip = icon.nextElementSibling;
      hideTooltip(tooltip);
    });
    showTooltip(tooltipUser);
    initial.style.color = "#7f9f7f";
  });

  circle.addEventListener('mouseleave', () => {
    hideTooltip(tooltipUser);
    initial.style.color = "#d4888d";
  });

  let hideTimeout;
  let activeTooltip;

  icons.forEach(function (icon) {
    icon.addEventListener("click", function (event) {
      event.stopPropagation();
      const appUrl = this.getAttribute("data-app-url");
      const appId = this.getAttribute("data-app-id");
      const modalId = this.getAttribute("data-modal-id");

      if (!appUrl && modalId) {
        openModal(document.getElementById(modalId));
      } else {
        if (!iframes[appId]) createIframe(appId, appUrl);
        showIframe(appId);
        document.querySelectorAll(".app-icon").forEach((icon) => {
          icon.style.color = "var(--bodyBg)";
        });
        this.style.color = "var(--button)";
      };
    });

    const tooltip = icon.nextElementSibling;
    [
      ['mouseenter', () => {
        clearTimeout(hideTimeout);
        if (activeTooltip && activeTooltip !== tooltip) {
          hideTooltip(activeTooltip);
        };
        const rect = icon.getBoundingClientRect();
        tooltip.style.top = `${rect.top + 26}px`;
        showTooltip(tooltip);
        activeTooltip = tooltip;
      }],
      ['mouseleave', () => {
        hideTimeout = setTimeout(() => {
          hideTooltip(tooltip);
          if (activeTooltip === tooltip) {
            activeTooltip = null;
          };
        }, 200);
      }],
    ].forEach(([event, listener]) => {
      icon.addEventListener(event, listener);
    });

    tooltip.addEventListener('mouseenter', () => {
      clearTimeout(hideTimeout);
    });

    tooltip.addEventListener('mouseleave', () => {
      hideTimeout = setTimeout(() => {
        hideTooltip(tooltip);
        if (activeTooltip === tooltip) {
          activeTooltip = null;
        };
      }, 200);
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
      let clDiv = document.getElementById(helpContentId).parentNode;
      const allContentElements = clDiv.querySelectorAll('.help-content');

      allContentElements.forEach(function (contentElement) {
        contentElement.style.display = "none";
      });

      let currentHelpTitle = button;
      while (currentHelpTitle && !currentHelpTitle.classList.contains('modal-left')) {
        currentHelpTitle = currentHelpTitle.parentNode;
        if (currentHelpTitle === document.body) {
          currentHelpTitle = null;
          break;
        }
      }

      currentHelpTitle.querySelectorAll('.title-menu').forEach(function (button) {
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
    if (visibleModal === null || visibleModal === undefined) return;
    const modalContent = visibleModal.querySelector("article");
    const closeButton = visibleModal.querySelector(".close-button");
    if (!modalContent.contains(event.target) || closeButton.contains(event.target)) {
      closeModal(visibleModal);
    }
  });

  const iframes = {};
  const rightPanel = document.getElementById('right-panel');

  function showIframe(appId) {
    document.querySelectorAll('.appFrame').forEach(iframe => {
      iframe.style.display = iframe.id === `appFrame-${appId}` ? 'block' : 'none';
    });
  };

  function createIframe(appId, appUrl) {
    const iframe = document.createElement('iframe');
    iframe.id = `appFrame-${appId}`;
    iframe.className = 'appFrame';
    iframe.src = appUrl;
    iframe.style.display = 'none';
    rightPanel.appendChild(iframe);
    iframes[appId] = iframe;
  };
});

function copyToClipboard(elem, text) {
  navigator.clipboard.writeText(text);
  elem.setAttribute("data-tooltip", "Copied");
  elem.setAttribute("data-placement", "bottom");
  setTimeout(() => {
    elem.removeAttribute("data-tooltip");
    elem.removeAttribute("data-placement");
  }, 500);
};
