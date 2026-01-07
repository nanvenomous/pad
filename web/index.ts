import htmx from "htmx.org";
import { dismissAlert } from './alert'
import { dialogEventHandler } from "./dialog";
import { applySavedTheme, persistTheme, updateThemeDisplay } from './theme'
import { initNoteEditors } from "./note_editor";

const w = window as any;
w.htmx = htmx;

w.dismissAlert = dismissAlert
w.dialogEventHandler = dialogEventHandler
w.applySavedTheme = applySavedTheme
w.persistTheme = persistTheme
w.updateThemeDisplay = updateThemeDisplay

const initEditorsOnce = () => initNoteEditors(document);
if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initEditorsOnce);
} else {
  initEditorsOnce();
}
htmx.onLoad((root) => {
  initNoteEditors(root);
});
