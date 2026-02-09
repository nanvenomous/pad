import { showAlert } from './alert';

export async function shareNote(id: string, title: string): Promise<void> {
  const shareUrl = window.location.href;
  
  // Check if Web Share API is available
  // In PWA or mobile browsers, navigator.share should be available
  if (navigator.share) {
    try {
      // Get the note content from the editor
      const noteContent = getNoteContent(id);
      
      // Construct share data
      const shareData: ShareData = {
        title: title || 'Note',
        text: noteContent || '',
        url: shareUrl
      };

      console.log('Attempting to share:', shareData);
      await navigator.share(shareData);
      console.log('Share successful');
      // Success - no need to show alert as the share sheet provides feedback
    } catch (error) {
      // User cancelled the share or an error occurred
      const errorName = (error as Error).name;
      console.error('Share error:', errorName, error);
      
      if (errorName !== 'AbortError') {
        // Real error (not user cancellation)
        showAlert(`Failed to share: ${errorName}`, 'error');
      }
    }
  } else {
    // Fallback for desktop: copy link to clipboard
    console.log('Web Share API not available, using clipboard fallback');
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
