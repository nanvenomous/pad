export function dismissAlert (toastID: string, buttonID: string, timeout: number) {
  const toastElement = document.getElementById(toastID);
  const dismissButton = document.getElementById(buttonID);
  if (toastElement && dismissButton) {
    dismissButton.addEventListener("click", () => {
      toastElement.remove();
    });

    setTimeout(function () {
      toastElement.remove();
    }, timeout * 1000);
  }
};

type AlertType = 'info' | 'success' | 'warning' | 'error';

const alertTypeClasses: Record<AlertType, string> = {
  info: 'alert-info',
  success: 'alert-success',
  warning: 'alert-warning',
  error: 'alert-error'
};

const alertIconClasses: Record<AlertType, string> = {
  info: 'ph-info',
  success: 'ph-check-circle',
  warning: 'ph-warning',
  error: 'ph-warning-circle'
};

export function showAlert(message: string, type: AlertType = 'info', timeout: number = 6) {
  const container = document.getElementById('toastContainer');
  if (!container) {
    console.warn('Toast container not found');
    return;
  }

  const id = generateId();
  const toastID = `toastNotif_${id}`;
  const buttonID = `toastDismissBtn_${id}`;

  const alertDiv = document.createElement('div');
  alertDiv.id = toastID;
  alertDiv.setAttribute('role', 'alert');
  alertDiv.className = `alert ${alertTypeClasses[type]} mt-4 pointer-events-auto`;

  alertDiv.innerHTML = `
    <i class="ph-light text-xl ${alertIconClasses[type]}"></i>
    <span>${escapeHtml(message)}</span>
    <button id="${buttonID}" class="btn btn-ghost btn-square cursor-pointer h-6 w-6">
      <i class="ph-light ph-x text-xl"></i>
    </button>
  `;

  container.appendChild(alertDiv);

  // Set up dismiss handlers
  dismissAlert(toastID, buttonID, timeout);
}

function generateId(): string {
  return Math.random().toString(36).substring(2, 15);
}

function escapeHtml(text: string): string {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}
