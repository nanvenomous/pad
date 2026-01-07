const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

let updatedTimer: number | undefined;

function formatAbsolute(date: Date): string {
  const month = MONTHS[date.getMonth()] ?? "Jan";
  return `${month} ${date.getDate()}, ${date.getFullYear()}`;
}

function humanizeUpdated(date: Date): string {
  const diffMs = Math.max(0, Date.now() - date.getTime());
  const minuteMs = 60 * 1000;
  const hourMs = 60 * minuteMs;
  const dayMs = 24 * hourMs;
  const weekMs = 7 * dayMs;

  if (diffMs < minuteMs) return "just now";
  if (diffMs < hourMs) return `${Math.floor(diffMs / minuteMs)}m ago`;
  if (diffMs < dayMs) return `${Math.floor(diffMs / hourMs)}h ago`;
  if (diffMs < weekMs) return `${Math.floor(diffMs / dayMs)}d ago`;
  return formatAbsolute(date);
}

function updateUpdatedLabels(root: ParentNode) {
  const labels = root.querySelectorAll<HTMLElement>("[data-updated-at]");
  labels.forEach((label) => {
    const raw = label.dataset.updatedAt;
    if (!raw) return;
    const updatedAt = new Date(raw);
    if (Number.isNaN(updatedAt.getTime())) return;
    label.textContent = humanizeUpdated(updatedAt);
  });
}

export function initUpdatedLabels(root: ParentNode) {
  updateUpdatedLabels(root);
  if (updatedTimer === undefined) {
    updatedTimer = window.setInterval(() => updateUpdatedLabels(document), 60 * 1000);
  }
}
