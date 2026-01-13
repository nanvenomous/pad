

export function dialogClose(dialogID: string) {
  const dialog = document.getElementById(dialogID) as HTMLDialogElement;
  if (dialog) {
    dialog.close();
  }
}

export function dialogEventHandler (dialogID: string) {
  const dialog = document.getElementById(dialogID) as HTMLDialogElement;
  if (dialog) {
    if (!dialog.dataset.dialogBound) {
      dialog.dataset.dialogBound = "true";
      dialog.addEventListener('click', (event: MouseEvent) => {
        if (event.target instanceof Element && event.target.id === dialogID) {
          dialog.close();
        }
      });
      dialog.addEventListener('cancel', (event: Event) => {
        event.preventDefault();
        dialog.close();
      });
    }
    if (!dialog.open) {
      requestAnimationFrame(() => {
        dialog.getBoundingClientRect();
        requestAnimationFrame(() => {
          if (!dialog.open) {
            dialog.showModal();
          }
        });
      });
    }
  } else {
    console.error("could not find dialog element with id: ", dialogID)
  }
}
