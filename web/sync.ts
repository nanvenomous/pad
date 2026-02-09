// Minimal JS: tell backend when app becomes visible or is reopened
// Backend handles all rendering and state management via HTMX

let lastHiddenTime = 0;
let lastSyncTime = 0;
let isInitialLoad = true;

function triggerSync() {
  const now = Date.now();
  const hiddenDuration = now - lastHiddenTime;
  
  // Get current note ID from the DOM
  const selectedNote = document.querySelector('[data-note-id]');
  const noteId = selectedNote?.getAttribute('data-note-id') || '';

  // Trigger HTMX GET request - backend will handle throttling
  const htmx = (window as any).htmx;
  if (htmx) {
    htmx.ajax('GET', `/notes/sync?id=${encodeURIComponent(noteId)}&hidden_ms=${hiddenDuration}`, {
      target: '#notesMain',
      swap: 'none' // Let server decide via hx-swap-oob
    });
  }
  
  lastSyncTime = now;
}

export function initSyncOnVisibility() {
  // Handle visibility changes (tab switching, backgrounding)
  document.addEventListener('visibilitychange', () => {
    if (document.hidden) {
      lastHiddenTime = Date.now();
      return;
    }
    
    // Skip initial load (handled by pageshow)
    if (isInitialLoad) {
      isInitialLoad = false;
      return;
    }

    triggerSync();
  });

  // Handle page show (critical for PWAs!)
  // Fires when:
  // - Page first loads
  // - PWA is reopened after being completely closed/killed
  // - Browser back/forward navigation (with bfcache)
  window.addEventListener('pageshow', (event) => {
    const now = Date.now();
    
    // On initial page load, just record the time
    if (isInitialLoad && !event.persisted) {
      lastHiddenTime = now;
      lastSyncTime = now;
      isInitialLoad = false;
      return;
    }
    
    // If page was restored from bfcache (back/forward)
    // or if it's been >2 seconds since last sync, trigger sync
    if (event.persisted || (now - lastSyncTime) > 2000) {
      // Set hidden time to simulate being away
      if (lastHiddenTime === 0) {
        lastHiddenTime = now - 10000; // Assume 10s away if unknown
      }
      triggerSync();
    }
  });

  // Handle focus event as backup (iOS Safari sometimes doesn't fire pageshow)
  window.addEventListener('focus', () => {
    const now = Date.now();
    
    // Only trigger if it's been >2 seconds since last sync
    // and we're not on initial load
    if (!isInitialLoad && (now - lastSyncTime) > 2000) {
      if (lastHiddenTime === 0) {
        lastHiddenTime = now - 5000; // Assume 5s away
      }
      triggerSync();
    }
  });
}
