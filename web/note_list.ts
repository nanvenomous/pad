import htmx from "htmx.org";
import { shareNote } from "./share";

const longPressDelayMs = 500;

function openActions(id: string, title: string, revision: string, x: number, y: number): void {
  const menu = document.getElementById("actionsMenu") as HTMLDetailsElement | null;
  if (!menu) {
    return;
  }
  const titleEl = menu.querySelector<HTMLElement>("[data-actions-title]");
  if (titleEl) {
    titleEl.textContent = title;
  }
  const idInput = menu.querySelector<HTMLInputElement>("[data-actions-id]");
  if (idInput) {
    idInput.value = id;
  }
  const revisionInput = menu.querySelector<HTMLInputElement>("[data-actions-revision]");
  if (revisionInput) {
    revisionInput.value = revision;
  }
  const moveButton = menu.querySelector<HTMLElement>("[data-actions-move]");
  if (moveButton) {
    moveButton.setAttribute("hx-get", `/notes/move-modal?id=${encodeURIComponent(id)}`);
    htmx.process(moveButton);
  }
  
  // Set up share button with note data
  const shareButton = menu.querySelector<HTMLElement>("[data-actions-share]");
  if (shareButton) {
    shareButton.dataset.shareId = id;
    shareButton.dataset.shareTitle = title;
    
    // Remove any existing click handler to avoid duplicates
    const oldHandler = (shareButton as any)._shareClickHandler;
    if (oldHandler) {
      shareButton.removeEventListener('click', oldHandler);
    }
    
    // Add direct click handler for immediate response
    const newHandler = (e: Event) => {
      e.preventDefault();
      e.stopPropagation();
      console.log('Direct share button handler triggered');
      shareNote(id, title);
      closeActionsMenu();
    };
    shareButton.addEventListener('click', newHandler);
    (shareButton as any)._shareClickHandler = newHandler;
  }
  
  menu.style.left = `${x}px`;
  menu.style.top = `${y}px`;
  menu.classList.remove("hidden");
  menu.classList.add("dropdown-open");
  menu.setAttribute("open", "");
}

export function initNoteList(root: ParentNode = document): void {
  const items = root.querySelectorAll<HTMLElement>("[data-note-item][data-note-id]");
  items.forEach((item) => {
    if (item.dataset.noteListReady === "true") {
      return;
    }
    item.dataset.noteListReady = "true";
    const id = item.dataset.noteId;
    if (!id) {
      return;
    }
    const title = item.dataset.noteTitle ?? "Note";
    const revision = item.dataset.noteRevision ?? "0";

    let timer: number | null = null;
    let longPressed = false;

    item.addEventListener("pointerdown", (event) => {
      if (event.pointerType === "mouse") {
        return;
      }
      longPressed = false;
      const point = clampPoint(event.clientX, event.clientY);
      if (timer) {
        window.clearTimeout(timer);
      }
      timer = window.setTimeout(() => {
        longPressed = true;
        openActions(id, title, revision, point.x, point.y);
      }, longPressDelayMs);
    });

    item.addEventListener("pointerup", () => {
      if (timer) {
        window.clearTimeout(timer);
        timer = null;
      }
    });

    item.addEventListener("pointercancel", () => {
      if (timer) {
        window.clearTimeout(timer);
        timer = null;
      }
    });

    item.addEventListener("click", (event) => {
      if (longPressed) {
        event.preventDefault();
        event.stopPropagation();
      }
    });

    item.addEventListener("contextmenu", (event) => {
      event.preventDefault();
      const point = clampPoint(event.clientX, event.clientY);
      openActions(id, title, revision, point.x, point.y);
    });
  });
}

export function initNoteActions(root: ParentNode = document): void {
  const buttons = root.querySelectorAll<HTMLElement>("[data-note-actions][data-note-id]");
  buttons.forEach((button) => {
    if (button.dataset.noteActionsReady === "true") {
      return;
    }
    button.dataset.noteActionsReady = "true";
    const id = button.dataset.noteId;
    if (!id) {
      return;
    }
    const title = button.dataset.noteTitle ?? "Note";
    const revision = button.dataset.noteRevision ?? "0";

    button.addEventListener("click", (event) => {
      event.preventDefault();
      const rect = button.getBoundingClientRect();
      const point = clampPoint(rect.right, rect.bottom);
      openActions(id, title, revision, point.x, point.y);
    });
  });

  if ((document.body as HTMLElement).dataset.actionsMenuReady === "true") {
    return;
  }
  (document.body as HTMLElement).dataset.actionsMenuReady = "true";

  document.addEventListener("click", (event) => {
    const target = event.target as HTMLElement | null;
    if (!target) {
      return;
    }
    
    // Handle share button click
    const shareButton = target.closest("[data-actions-share]") as HTMLElement | null;
    if (shareButton) {
      console.log('Share button clicked');
      const id = shareButton.dataset.shareId;
      const title = shareButton.dataset.shareTitle;
      console.log('Share data:', { id, title });
      if (id && title) {
        console.log('Calling shareNote function');
        shareNote(id, title);
      } else {
        console.error('Missing share data:', { id, title });
      }
      return;
    }
    
    if (target.closest("[data-actions-close]")) {
      closeActionsMenu();
      return;
    }
    const menu = document.getElementById("actionsMenu");
    if (menu && !menu.contains(target) && !target.closest("[data-note-actions]")) {
      closeActionsMenu();
    }
  });

  document.addEventListener(
    "pointerdown",
    (event) => {
      const target = event.target as HTMLElement | null;
      if (!target) {
        return;
      }
      if (target.closest("#actionsMenu") || target.closest("[data-note-actions]")) {
        return;
      }
      closeActionsMenu();
    },
    true,
  );

  document.addEventListener("keydown", (event) => {
    if (event.key !== "Escape") {
      return;
    }
    closeActionsMenu();
  });
}

function closeActionsMenu(): void {
  const menu = document.getElementById("actionsMenu") as HTMLDetailsElement | null;
  if (!menu) {
    return;
  }
  menu.classList.add("hidden");
  menu.removeAttribute("open");
  menu.classList.remove("dropdown-open");
}

function clampPoint(x: number, y: number): { x: number; y: number } {
  const margin = 12;
  const menuWidth = 240;
  const menuHeight = 220; // Increased to accommodate share button
  const maxX = Math.max(margin, window.innerWidth - menuWidth - margin);
  const maxY = Math.max(margin, window.innerHeight - menuHeight - margin);
  const nextX = Math.min(Math.max(margin, x), maxX);
  const nextY = Math.min(Math.max(margin, y), maxY);
  return { x: nextX, y: nextY };
}
