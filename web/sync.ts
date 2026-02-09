// Minimal JS: tell backend when app becomes visible or is reopened
// Backend handles all rendering and state management via HTMX

let lastHiddenTime = 0;
let lastSyncTime = 0;
let hasLoadedOnce = false;

function triggerSync() {
  const now = Date.now();
  const hiddenDuration = now - lastHiddenTime;
  
  console.log('[Pad Sync] Triggering sync - hidden for', hiddenDuration, 'ms');
  
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
  // Initialize on first load
  const now = Date.now();
  lastHiddenTime = now;
  lastSyncTime = now;
  
  // Handle visibility changes (tab switching, backgrounding, PWA minimize/restore)
  document.addEventListener('visibilitychange', () => {
    const now = Date.now();
    
    if (document.hidden) {
      lastHiddenTime = now;
      console.log('[Pad Sync] App hidden at', lastHiddenTime);
      return;
    }
    
    const hiddenDuration = now - lastHiddenTime;
    const timeSinceLastSync = now - lastSyncTime;
    
    console.log('[Pad Sync] App visible - hasLoadedOnce:', hasLoadedOnce, 
                'hidden:', hiddenDuration, 'ms, time since last sync:', timeSinceLastSync, 'ms');
    
    // Sync if:
    // 1. Not initial load, AND
    // 2. Either hidden for >1s OR it's been >5s since last sync
    if (hasLoadedOnce && (hiddenDuration > 1000 || timeSinceLastSync > 5000)) {
      triggerSync();
    } else {
      console.log('[Pad Sync] Skipping sync - too recent');
    }
  });

  // Handle page show (critical for PWAs!)
  // Fires when:
  // - Page first loads
  // - PWA is reopened after being completely closed/killed
  // - Browser back/forward navigation (with bfcache)
  window.addEventListener('pageshow', (event) => {
    const now = Date.now();
    
    console.log('[Pad Sync] pageshow - persisted:', event.persisted, 'hasLoadedOnce:', hasLoadedOnce);
    
    // On initial page load, just mark as loaded
    if (!hasLoadedOnce && !event.persisted) {
      hasLoadedOnce = true;
      return;
    }
    
    // If page was restored from bfcache (back/forward)
    // or if it's been >2 seconds since last sync, trigger sync
    if (event.persisted || (now - lastSyncTime) > 2000) {
      // Set hidden time to simulate being away
      if (lastHiddenTime === 0 || (now - lastHiddenTime) < 1000) {
        lastHiddenTime = now - 10000; // Assume 10s away if unknown
      }
      triggerSync();
    }
  });

  // Handle focus event as backup (iOS Safari sometimes doesn't fire pageshow)
  window.addEventListener('focus', () => {
    const now = Date.now();
    
    // Only trigger if it's been >2 seconds since last sync
    // and we've loaded once already
    if (hasLoadedOnce && (now - lastSyncTime) > 2000) {
      if (lastHiddenTime === 0 || (now - lastHiddenTime) < 1000) {
        lastHiddenTime = now - 5000; // Assume 5s away
      }
      triggerSync();
    }
  });
}
