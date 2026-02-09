package handle

import "sync"

// ResetNotesStoreForTesting resets the singleton store - ONLY for testing
func ResetNotesStoreForTesting() {
	notesStore = nil
	notesStoreErr = nil
	notesStoreOnce = sync.Once{}

	// Also reset realtime hub
	notesRealtimeOnce = sync.Once{}
	notesHubInstance = nil
}
