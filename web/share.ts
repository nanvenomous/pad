import { showAlert } from './alert';

export async function shareNote(id: string, title: string): Promise<void> {
  const shareUrl = window.location.href;
  
  // Check if Web Share API is available (typically mobile)
  if (navigator.share && isMobileDevice()) {
    try {
      // Get the note content from the editor
      const noteContent = getNoteContent(id);
      
      // Construct share data
      const shareData: ShareData = {
        title: title || 'Note',
        text: noteContent || '',
        url: shareUrl
      };

      await navigator.share(shareData);
      // Success - no need to show alert as the share sheet provides feedback
    } catch (error) {
      // User cancelled the share or an error occurred
      if ((error as Error).name !== 'AbortError') {
        console.error('Error sharing:', error);
        showAlert('Failed to share note', 'error');
      }
    }
  } else {
    // Fallback for desktop: copy link to clipboard
    try {
      await navigator.clipboard.writeText(shareUrl);
      showAlert('Link copied to clipboard', 'success');
    } catch (error) {
      console.error('Failed to copy to clipboard:', error);
      showAlert('Failed to copy link to clipboard', 'error');
    }
  }
}

function getNoteContent(id: string): string {
  // Try to get content from the current editor if it matches the note ID
  const noteIdInput = document.getElementById('noteId') as HTMLInputElement | null;
  if (noteIdInput && noteIdInput.value === id) {
    const textarea = document.querySelector<HTMLTextAreaElement>('[data-note-body]');
    if (textarea) {
      return textarea.value;
    }
  }
  
  // Fallback: return empty if content can't be found
  return '';
}

function isMobileDevice(): boolean {
  // Check if the device is likely mobile based on user agent and touch support
  return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent) ||
    (!!navigator.maxTouchPoints && navigator.maxTouchPoints > 2);
}
