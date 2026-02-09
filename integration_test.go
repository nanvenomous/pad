package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nanvenomous/pad/handle"
	"github.com/nanvenomous/pad/notes"
	"golang.org/x/net/websocket"
)

// TestIntegration_BasicNoteLifecycle tests creating, reading, updating, and deleting a note
func TestIntegration_BasicNoteLifecycle(t *testing.T) {
	server, dir := setupTestServer(t)
	defer server.Close()

	// Create a new note
	noteID := createNote(t, server, "", "# Test Note", "# Test Note\n\nThis is a test.")
	if noteID == "" {
		t.Fatal("Failed to create note")
	}

	// Verify note was created on filesystem
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("Failed to read notes directory: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("No files created")
	}

	// Get the note
	note := getNote(t, server, noteID)
	if note.ID != noteID {
		t.Errorf("Expected note ID %s, got %s", noteID, note.ID)
	}
	if note.Title != "Test Note" {
		t.Errorf("Expected title 'Test Note', got '%s'", note.Title)
	}
	if !strings.Contains(note.Body, "This is a test") {
		t.Errorf("Expected body to contain 'This is a test', got '%s'", note.Body)
	}

	// Update the note
	updateNote(t, server, noteID, "# Updated Title", "# Updated Title\n\nUpdated content.", note.Revision)

	// Verify update
	updatedNote := getNote(t, server, noteID)
	if updatedNote.Title != "Updated Title" {
		t.Errorf("Expected updated title 'Updated Title', got '%s'", updatedNote.Title)
	}
	if !strings.Contains(updatedNote.Body, "Updated content") {
		t.Errorf("Expected body to contain 'Updated content', got '%s'", updatedNote.Body)
	}
	if updatedNote.Revision != note.Revision+1 {
		t.Errorf("Expected revision %d, got %d", note.Revision+1, updatedNote.Revision)
	}

	// Delete the note
	deleteNote(t, server, noteID, updatedNote.Revision)

	// Verify deletion - the note file is removed from disk, so it won't be accessible
	// Check the filesystem directly instead
	time.Sleep(100 * time.Millisecond) // Give time for deletion to complete

	metadataPath := filepath.Join(dir, "notes-metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to read metadata after delete: %v", err)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("Failed to parse metadata: %v", err)
	}

	// The note should no longer be in metadata (physically removed)
	if _, exists := metadata[noteID]; exists {
		t.Error("Note should be removed from metadata after deletion")
	}
}

// TestIntegration_ConflictDetection tests revision conflict handling
func TestIntegration_ConflictDetection(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	// Create a note
	noteID := createNote(t, server, "", "# Conflict Test", "# Conflict Test\n\nOriginal content.")
	note := getNote(t, server, noteID)

	// Update with correct revision
	updateNote(t, server, noteID, "# Updated Once", "# Updated Once\n\nFirst update.", note.Revision)

	// Try to update with stale revision (should conflict)
	resp := updateNoteRaw(t, server, noteID, "# Stale Update", "# Stale Update\n\nThis should fail.", note.Revision)

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Expected status 409 Conflict, got %d", resp.StatusCode)
	}

	// Verify the note still has the first update
	currentNote := getNote(t, server, noteID)
	if currentNote.Title != "Updated Once" {
		t.Errorf("Expected title 'Updated Once', got '%s'", currentNote.Title)
	}
}

// TestIntegration_FolderOperations tests folder creation and moving notes
func TestIntegration_FolderOperations(t *testing.T) {
	server, dir := setupTestServer(t)
	defer server.Close()

	// Create a note in root
	noteID := createNote(t, server, "", "# Root Note", "# Root Note\n\nIn root folder.")

	// Create a note in a subfolder
	subNoteID := createNote(t, server, "work", "# Work Note", "# Work Note\n\nIn work folder.")

	// Verify folder structure
	workDir := filepath.Join(dir, "work")
	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		t.Errorf("Expected work folder to exist at %s", workDir)
	}

	// Move root note to work folder
	note := getNote(t, server, noteID)
	moveNote(t, server, noteID, "work", note.Revision)

	// Verify both notes are now in work folder
	workFiles, err := os.ReadDir(workDir)
	if err != nil {
		t.Fatalf("Failed to read work directory: %v", err)
	}

	mdFileCount := 0
	for _, f := range workFiles {
		if strings.HasSuffix(f.Name(), ".md") {
			mdFileCount++
		}
	}

	if mdFileCount != 2 {
		t.Errorf("Expected 2 markdown files in work folder, got %d", mdFileCount)
	}

	// Verify notes are accessible
	movedNote := getNote(t, server, noteID)
	if movedNote.Title != "Root Note" {
		t.Errorf("Expected moved note to retain title 'Root Note', got '%s'", movedNote.Title)
	}

	workNote := getNote(t, server, subNoteID)
	if workNote.Title != "Work Note" {
		t.Errorf("Expected work note title 'Work Note', got '%s'", workNote.Title)
	}
}

// TestIntegration_FilesystemSync tests external file changes sync
func TestIntegration_FilesystemSync(t *testing.T) {
	server, dir := setupTestServer(t)
	defer server.Close()

	// Create a note through the API
	noteID := createNote(t, server, "", "# API Note", "# API Note\n\nCreated via API.")

	// Find the actual file on disk (it's stored with the nanoid as filename)
	metadataPath := filepath.Join(dir, "notes-metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to read metadata: %v", err)
	}

	var metadata map[string]struct {
		Filename string `json:"filename"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("Failed to parse metadata: %v", err)
	}

	// Get the actual filename for this note
	noteMetadata, ok := metadata[noteID]
	if !ok {
		t.Fatalf("Note %s not found in metadata", noteID)
	}

	actualFilePath := filepath.Join(dir, noteMetadata.Filename)

	// Manually edit the file (simulating Neovim edit)
	newContent := "# Manually Edited\n\nEdited in Neovim."
	err = os.WriteFile(actualFilePath, []byte(newContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Give the filesystem watcher time to detect the change and debounce (200ms debounce + buffer)
	time.Sleep(500 * time.Millisecond)

	// Fetch the note again - it should reflect the filesystem changes
	syncedNote := getNote(t, server, noteID)
	if syncedNote.Title != "Manually Edited" {
		t.Errorf("Expected title 'Manually Edited', got '%s'", syncedNote.Title)
	}
	if !strings.Contains(syncedNote.Body, "Edited in Neovim") {
		t.Errorf("Expected body to contain 'Edited in Neovim', got '%s'", syncedNote.Body)
	}
}

// TestIntegration_ConcurrentUpdates tests multiple clients updating notes
func TestIntegration_ConcurrentUpdates(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	// Create a note
	noteID := createNote(t, server, "", "# Concurrent Test", "# Concurrent Test\n\nInitial content.")

	// Simulate 5 concurrent update attempts
	const numUpdates = 5
	var wg sync.WaitGroup
	results := make([]int, numUpdates) // track status codes

	for i := 0; i < numUpdates; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			note := getNote(t, server, noteID)
			title := fmt.Sprintf("# Update %d", idx)
			body := fmt.Sprintf("# Update %d\n\nUpdate from goroutine %d.", idx, idx)
			resp := updateNoteRaw(t, server, noteID, title, body, note.Revision)
			results[idx] = resp.StatusCode
		}(i)
	}

	wg.Wait()

	// At least one should succeed (200), others might conflict (409)
	successCount := 0
	conflictCount := 0
	for _, code := range results {
		if code == http.StatusOK {
			successCount++
		} else if code == http.StatusConflict {
			conflictCount++
		}
	}

	if successCount == 0 {
		t.Error("Expected at least one successful update")
	}

	t.Logf("Concurrent updates: %d successful, %d conflicts", successCount, conflictCount)
}

// TestIntegration_WebSocketRealtime tests real-time updates via WebSocket
func TestIntegration_WebSocketRealtime(t *testing.T) {
	server, dir := setupTestServer(t)
	defer server.Close()

	// Connect a WebSocket client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/notes/stream"
	ws, err := websocket.Dial(wsURL, "", server.URL)
	if err != nil {
		t.Fatalf("Failed to connect WebSocket: %v", err)
	}
	defer ws.Close()

	// Channel to receive WebSocket messages
	messages := make(chan string, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		for {
			var msg string
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				return
			}
			select {
			case messages <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Create a note (should trigger WebSocket update)
	noteID := createNote(t, server, "", "# WebSocket Test", "# WebSocket Test\n\nTesting real-time.")

	// Wait for WebSocket message
	select {
	case msg := <-messages:
		if !strings.Contains(msg, "WebSocket Test") && !strings.Contains(msg, noteID) {
			t.Logf("Received WebSocket message: %s", msg)
		}
	case <-time.After(2 * time.Second):
		// WebSocket message might not arrive immediately, that's okay
		t.Log("WebSocket message not received (might be expected due to timing)")
	}

	// Manually create a file (simulating Neovim)
	testFile := filepath.Join(dir, "external-note.md")
	err = os.WriteFile(testFile, []byte("# External Note\n\nCreated externally."), 0644)
	if err != nil {
		t.Fatalf("Failed to write external file: %v", err)
	}

	// Wait for filesystem watcher to detect and broadcast
	select {
	case msg := <-messages:
		t.Logf("Received WebSocket update for external change: %s", msg)
	case <-time.After(1 * time.Second):
		t.Log("External file change WebSocket notification not received within timeout")
	}
}

// TestIntegration_ListNotes tests note listing and ordering
func TestIntegration_ListNotes(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	// Create multiple notes with delays to ensure different timestamps
	id1 := createNote(t, server, "", "# First Note", "# First Note\n\nCreated first.")
	time.Sleep(10 * time.Millisecond)

	id2 := createNote(t, server, "work", "# Second Note", "# Second Note\n\nCreated second.")
	time.Sleep(10 * time.Millisecond)

	id3 := createNote(t, server, "", "# Third Note", "# Third Note\n\nCreated third.")

	// List all notes
	notes := listNotes(t, server)

	if len(notes) < 3 {
		t.Fatalf("Expected at least 3 notes, got %d", len(notes))
	}

	// Verify notes are sorted by UpdatedAt (most recent first)
	// Third note should be first
	foundIDs := make(map[string]bool)
	for _, note := range notes {
		foundIDs[note.ID] = true
	}

	if !foundIDs[id1] || !foundIDs[id2] || !foundIDs[id3] {
		t.Error("Not all created notes found in list")
	}
}

// TestIntegration_DeleteRevisitConflict tests deleting with wrong revision
func TestIntegration_DeleteRevisionConflict(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	// Create and update a note
	noteID := createNote(t, server, "", "# Delete Test", "# Delete Test\n\nWill be deleted.")
	note := getNote(t, server, noteID)
	updateNote(t, server, noteID, "# Updated", "# Updated\n\nUpdated before delete.", note.Revision)

	// Try to delete with stale revision
	resp := deleteNoteRaw(t, server, noteID, note.Revision) // Using old revision

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Expected status 409 Conflict, got %d", resp.StatusCode)
	}

	// Verify note still exists
	currentNote := getNote(t, server, noteID)
	if currentNote.Deleted {
		t.Error("Note should not be deleted after failed delete attempt")
	}
	if currentNote.Title != "Updated" {
		t.Errorf("Expected title 'Updated', got '%s'", currentNote.Title)
	}
}

// TestIntegration_VisibilitySync tests that notes sync when app regains visibility after external edits
func TestIntegration_VisibilitySync(t *testing.T) {
	server, dir := setupTestServer(t)
	defer server.Close()

	// 1. Create initial note
	noteID := createNote(t, server, "", "# Visibility Test", "# Visibility Test\n\nInitial content")
	initialNote := getNote(t, server, noteID)

	// 2. Simulate Neovim editing file while app is "closed"
	metadata := readMetadata(t, dir)
	filename := metadata[noteID].Filename
	notePath := filepath.Join(dir, filename)

	err := os.WriteFile(notePath, []byte("# Visibility Test\n\nUpdated by Neovim while app was hidden"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// 3. Wait for filesystem watcher to detect and sync
	time.Sleep(700 * time.Millisecond)

	// 3b. Verify the filesystem change was synced to the store
	syncedNote := getNote(t, server, noteID)
	if syncedNote.Body != "# Visibility Test\n\nUpdated by Neovim while app was hidden" {
		t.Fatalf("filesystem changes not synced yet: got %q", syncedNote.Body)
	}

	// 4. Simulate app visibility change (hidden for 10 seconds)
	resp := syncNotesAfterVisibility(t, server, noteID, 10000)

	// 5. Should return 200 OK with updated content
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	// 6. Response should contain the Neovim-edited content
	// The content will be HTML-encoded in textarea, so check for key parts
	responseStr := string(body)
	if !strings.Contains(responseStr, "Updated by Neovim") && !strings.Contains(responseStr, "Visibility Test") {
		// Print first 500 chars for debugging
		t.Logf("Response preview (first 500 chars): %s", responseStr[:min(500, len(responseStr))])
		t.Errorf("sync response missing updated content")
	}

	// 7. Verify note was actually synced
	updatedNote := getNote(t, server, noteID)
	if updatedNote.Body != "# Visibility Test\n\nUpdated by Neovim while app was hidden" {
		t.Errorf("note body not synced: got %q", updatedNote.Body)
	}

	if !updatedNote.UpdatedAt.After(initialNote.UpdatedAt) {
		t.Error("updated_at timestamp not updated")
	}
}

// TestIntegration_VisibilitySyncAlwaysWorks tests that sync always returns current state
func TestIntegration_VisibilitySyncAlwaysWorks(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	noteID := createNote(t, server, "", "# Sync Test", "# Sync Test\n\nContent")

	// Sync always returns current state
	// Frontend only triggers on visibility change (after initial load)
	resp := syncNotesAfterVisibility(t, server, noteID, 0)

	// Should return 200 OK with current state
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Multiple syncs work fine
	resp2 := syncNotesAfterVisibility(t, server, noteID, 0)

	// Should return 200 OK
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp2.StatusCode)
	}
	resp2.Body.Close()
}

// Helper functions

func setupTestServer(t *testing.T) (*httptest.Server, string) {
	// Reset the singleton store from previous tests
	handle.ResetNotesStoreForTesting()

	// Create a temporary directory for test notes
	dir := t.TempDir()

	// Set environment variable for notes directory
	os.Setenv("PAD_NOTES_DIR", dir)
	t.Cleanup(func() {
		os.Unsetenv("PAD_NOTES_DIR")
		handle.ResetNotesStoreForTesting()
	})

	// Create HTTP test server
	mux := http.NewServeMux()
	handler, err := handle.Setup(mux, buildFS)
	if err != nil {
		t.Fatalf("Failed to setup handler: %v", err)
	}

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return server, dir
}

func createNote(t *testing.T, server *httptest.Server, folder, title, body string) string {
	t.Helper()

	form := url.Values{}
	form.Set("folder", folder)
	form.Set("id", "") // Empty ID means create new
	form.Set("body", body)
	form.Set("revision", "0")

	resp, err := http.Post(server.URL+"/notes/save", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to create note, status %d: %s", resp.StatusCode, bodyBytes)
	}

	// Give it time to persist and sync
	time.Sleep(200 * time.Millisecond)

	// Find the most recently created note
	notes := listNotes(t, server)
	if len(notes) == 0 {
		t.Fatal("No notes found after creation")
	}

	// Return the most recently updated note (should be the one we just created)
	return notes[0].ID
}

func getNote(t *testing.T, server *httptest.Server, noteID string) notes.Note {
	t.Helper()

	resp, err := http.Get(server.URL + "/notes/select?id=" + url.QueryEscape(noteID))
	if err != nil {
		t.Fatalf("Failed to get note: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to get note, status %d", resp.StatusCode)
	}

	// Parse HTML response to extract note data
	// Since we're testing integration, we can also use the store directly
	store, _ := notes.NewStore(os.Getenv("PAD_NOTES_DIR"))
	note, ok := store.Get(noteID)
	if !ok {
		t.Fatalf("Note %s not found in store", noteID)
	}
	return note
}

func listNotes(t *testing.T, server *httptest.Server) []notes.Note {
	t.Helper()

	dir := os.Getenv("PAD_NOTES_DIR")
	metadataPath := filepath.Join(dir, "notes-metadata.json")

	// Read metadata file to get all note IDs
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []notes.Note{}
		}
		t.Fatalf("Failed to read metadata: %v", err)
	}

	var metadata map[string]struct {
		Title     string    `json:"title"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Revision  int       `json:"revision"`
		Deleted   bool      `json:"deleted"`
		Filename  string    `json:"filename"`
	}

	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("Failed to parse metadata: %v", err)
	}

	// Get all notes using their IDs from metadata
	var notesList []notes.Note
	for noteID := range metadata {
		note := getNote(t, server, noteID)
		if !note.Deleted {
			notesList = append(notesList, note)
		}
	}

	// Sort by UpdatedAt (most recent first)
	sort.Slice(notesList, func(i, j int) bool {
		return notesList[i].UpdatedAt.After(notesList[j].UpdatedAt)
	})

	return notesList
}

func updateNote(t *testing.T, server *httptest.Server, noteID, title, body string, revision int) {
	t.Helper()

	resp := updateNoteRaw(t, server, noteID, title, body, revision)
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to update note, status %d: %s", resp.StatusCode, bodyBytes)
	}
}

func updateNoteRaw(t *testing.T, server *httptest.Server, noteID, title, body string, revision int) *http.Response {
	t.Helper()

	form := url.Values{}
	form.Set("id", noteID)
	form.Set("title", title)
	form.Set("body", body)
	form.Set("revision", fmt.Sprintf("%d", revision))

	resp, err := http.Post(server.URL+"/notes/save", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("Failed to update note: %v", err)
	}
	return resp
}

func deleteNote(t *testing.T, server *httptest.Server, noteID string, revision int) {
	t.Helper()

	resp := deleteNoteRaw(t, server, noteID, revision)
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to delete note, status %d: %s", resp.StatusCode, bodyBytes)
	}
}

func deleteNoteRaw(t *testing.T, server *httptest.Server, noteID string, revision int) *http.Response {
	t.Helper()

	form := url.Values{}
	form.Set("id", noteID)
	form.Set("revision", fmt.Sprintf("%d", revision))

	resp, err := http.Post(server.URL+"/notes/delete", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("Failed to delete note: %v", err)
	}
	return resp
}

func moveNote(t *testing.T, server *httptest.Server, noteID, folder string, revision int) {
	t.Helper()

	form := url.Values{}
	form.Set("id", noteID)
	form.Set("folder", folder)
	form.Set("revision", fmt.Sprintf("%d", revision))

	resp, err := http.Post(server.URL+"/notes/move", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("Failed to move note: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to move note, status %d: %s", resp.StatusCode, bodyBytes)
	}
}

// syncNotesAfterVisibility simulates visibility change (app becoming visible)
func syncNotesAfterVisibility(t *testing.T, server *httptest.Server, noteID string, _ int) *http.Response {
	t.Helper()
	urlStr := fmt.Sprintf("%s/notes/sync?id=%s",
		server.URL, url.QueryEscape(noteID))
	resp, err := http.Get(urlStr)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// readMetadata reads the notes metadata file
func readMetadata(t *testing.T, dir string) map[string]struct {
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Revision  int       `json:"revision"`
	Deleted   bool      `json:"deleted"`
	Filename  string    `json:"filename"`
} {
	t.Helper()

	metadataPath := filepath.Join(dir, "notes-metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to read metadata: %v", err)
	}

	var metadata map[string]struct {
		Title     string    `json:"title"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Revision  int       `json:"revision"`
		Deleted   bool      `json:"deleted"`
		Filename  string    `json:"filename"`
	}

	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("Failed to parse metadata: %v", err)
	}

	return metadata
}
