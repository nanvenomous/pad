import { showAlert } from './alert';

export async function shareNote(id: string, title: string): Promise<void> {
  const shareUrl = window.location.href;
  
  // Check if Web Share API is available
  // In PWA or mobile browsers, navigator.share should be available
  if (navigator.share) {
    try {
      // Construct share data - just share the link, not the content
      const shareData: ShareData = {
        title: title || 'Note',
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
