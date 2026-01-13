import htmx from "htmx.org";
import "htmx-ext-ws";
import { dismissAlert } from './alert'
import { dialogClose, dialogEventHandler } from "./dialog";
import { applySavedTheme, persistTheme, updateThemeDisplay } from './theme'
import { initNoteEditors } from "./note_editor";
import { initNoteActions, initNoteList } from "./note_list";
import { initUpdatedLabels } from "./updated_time";

const w = window as any;
w.htmx = htmx;
if (!(globalThis as any).htmx) {
  (globalThis as any).htmx = htmx;
}

w.dismissAlert = dismissAlert
w.dialogEventHandler = dialogEventHandler
w.dialogClose = dialogClose
w.applySavedTheme = applySavedTheme
w.persistTheme = persistTheme
w.updateThemeDisplay = updateThemeDisplay

const initEditorsOnce = () => {
  initNoteEditors(document);
  initNoteActions(document);
  initNoteList(document);
  initUpdatedLabels(document);
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
