import htmx from "htmx.org";
import "htmx-ext-ws";
import { dismissAlert, showAlert } from './alert'
import { dialogClose, dialogEventHandler } from "./dialog";
import { applySavedTheme, persistTheme, updateThemeDisplay } from './theme'
import { initNoteEditors } from "./note_editor";
import { initNoteActions, initNoteList } from "./note_list";
import { initUpdatedLabels } from "./updated_time";
import { initSyncOnVisibility } from "./sync";
import { shareNote } from "./share";

const w = window as any;
w.htmx = htmx;
if (!(globalThis as any).htmx) {
  (globalThis as any).htmx = htmx;
}

// Register service worker for PWA
if ('serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker
      .register('/service-worker.js')
      .then((registration) => {
        console.log('Service Worker registered:', registration.scope);
      })
      .catch((error) => {
        console.log('Service Worker registration failed:', error);
      });
  });
}

w.dismissAlert = dismissAlert
w.showAlert = showAlert
w.dialogEventHandler = dialogEventHandler
w.dialogClose = dialogClose
w.applySavedTheme = applySavedTheme
w.persistTheme = persistTheme
w.updateThemeDisplay = updateThemeDisplay
w.shareNote = shareNote

const initEditorsOnce = () => {
  initNoteEditors(document);
  initNoteActions(document);
  initNoteList(document);
  initUpdatedLabels(document);
  initSyncOnVisibility();
};
if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initEditorsOnce);
} else {
  initEditorsOnce();
}
htmx.onLoad((root) => {
  initNoteEditors(root);
  initNoteActions(root);
  initNoteList(root);
  initUpdatedLabels(root);
});
