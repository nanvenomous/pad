// Minimal JS: tell backend when app becomes visible
// Backend handles all rendering and state management via HTMX

let hasLoadedOnce = false;

function triggerSync() {
  // Get current note ID from the DOM
  const selectedNote = document.querySelector('[data-note-id]');
  const noteId = selectedNote?.getAttribute('data-note-id') || '';

  // Trigger HTMX GET request to refresh state
  const htmx = (window as any).htmx;
  if (htmx) {
    htmx.ajax('GET', `/notes/sync?id=${encodeURIComponent(noteId)}`, {
      target: '#notesMain',
      swap: 'none' // Let server decide via hx-swap-oob
    });
  }
}

export function initSyncOnVisibility() {
  // Handle visibility changes - sync whenever app becomes visible
  document.addEventListener('visibilitychange', () => {
    // Skip if app is being hidden
    if (document.hidden) {
      return;
    }
    
    // Skip initial page load
    if (!hasLoadedOnce) {
      hasLoadedOnce = true;
      return;
    }
    
    // App became visible - refresh state
    triggerSync();
  });
}
