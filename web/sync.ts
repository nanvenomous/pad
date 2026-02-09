// Minimal JS: just tell backend when app becomes visible
// Backend handles all rendering and state management via HTMX

export function initSyncOnVisibility() {
  let lastHiddenTime = 0;

  document.addEventListener('visibilitychange', () => {
    if (document.hidden) {
      lastHiddenTime = Date.now();
      return;
    }

    // App became visible
    const hiddenDuration = Date.now() - lastHiddenTime;

    // Get current note ID from the DOM
    const selectedNote = document.querySelector('[data-note-id]');
    const noteId = selectedNote?.getAttribute('data-note-id') || '';

    // Trigger HTMX GET request - backend will handle everything including throttling
    const htmx = (window as any).htmx;
    if (htmx) {
      htmx.ajax('GET', `/notes/sync?id=${encodeURIComponent(noteId)}&hidden_ms=${hiddenDuration}`, {
        target: '#notesMain',
        swap: 'none' // Let server decide via hx-swap-oob
      });
    }
  });
}
