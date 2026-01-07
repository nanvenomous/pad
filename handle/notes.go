package handle

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/a-h/templ"
	"github.com/nanvenomous/pad/notes"
	"github.com/nanvenomous/pad/ui"
)

const defaultNotesDir = "/tmp/pad"

var (
	notesStore     *notes.Store
	notesStoreErr  error
	notesStoreOnce sync.Once
)

func getNotesStore() (*notes.Store, error) {
	notesStoreOnce.Do(func() {
		dir := os.Getenv("PAD_NOTES_DIR")
		if dir == "" {
			dir = defaultNotesDir
		}
		notesStore, notesStoreErr = notes.NewStore(dir)
	})
	return notesStore, notesStoreErr
}

func init() {
	setupFuncs = append(setupFuncs, func(mux *http.ServeMux) {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "" || r.URL.Path == "/" {
				NotesPageHandler(w, r)
				return
			}
			serveResourceCachedETag(w, r, getBundledFile)
		})

		mux.HandleFunc("/notes/select", NotesSelectHandler)
		mux.HandleFunc("/notes/new", NotesNewHandler)
		mux.HandleFunc("/notes/save", NotesSaveHandler)
		mux.HandleFunc("/notes/delete", NotesDeleteHandler)
	})
}

func NotesPageHandler(w http.ResponseWriter, r *http.Request) {
	mainProps, err := buildNotesMainProps(r.URL.Query().Get("id"), false)
	if err != nil {
		errorHTTP(w, http.StatusInternalServerError, err)
		return
	}

	stts, err := render(w, r, ui.NotesPage(mainProps))
	if err != nil {
		errorHTTP(w, stts, err)
	}
}

func NotesSelectHandler(w http.ResponseWriter, r *http.Request) {
	mainProps, err := buildNotesMainProps(r.URL.Query().Get("id"), false)
	if err != nil {
		errorHTTP(w, http.StatusInternalServerError, err)
		return
	}

	stts, err := render(w, r, ui.NotesMain(mainProps, false))
	if err != nil {
		errorHTTP(w, stts, err)
	}
}

func NotesNewHandler(w http.ResponseWriter, r *http.Request) {
	mainProps, err := buildNotesMainProps("", true)
	if err != nil {
		errorHTTP(w, http.StatusInternalServerError, err)
		return
	}

	stts, err := render(w, r, ui.NotesMain(mainProps, false))
	if err != nil {
		errorHTTP(w, stts, err)
	}
}

func NotesSaveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	store, err := getNotesStore()
	if err != nil {
		errorHTTP(w, http.StatusInternalServerError, err)
		return
	}

	if err := r.ParseForm(); err != nil {
		errorHTTP(w, http.StatusBadRequest, err)
		return
	}

	time.Sleep(250 * time.Millisecond)

	id := strings.TrimSpace(r.FormValue("id"))
	body := r.FormValue("body")
	revision, _ := strconv.Atoi(r.FormValue("revision"))

	title := notes.NormalizeTitleFromBody(body)

	var note notes.Note
	if id == "" {
		note, err = store.Create(title, body)
		if err != nil {
			errorHTTP(w, http.StatusInternalServerError, err)
			return
		}
	} else {
		note, err = store.Update(id, title, body, revision)
		if err != nil && err != notes.ErrConflict && err != notes.ErrNotFound {
			errorHTTP(w, http.StatusInternalServerError, err)
			return
		}
		if err == notes.ErrConflict {
			w.WriteHeader(http.StatusConflict)
		}
		if err == notes.ErrNotFound {
			w.WriteHeader(http.StatusNotFound)
		}
	}

	saveErr := err
	selectedID := note.ID
	if saveErr == notes.ErrNotFound {
		selectedID = ""
	}

	mainProps, err := buildNotesMainProps(selectedID, false)
	if err != nil {
		errorHTTP(w, http.StatusInternalServerError, err)
		return
	}

	useHX := r.Header.Get("HX-Request") == "true"
	hasConflict := saveErr == notes.ErrConflict || saveErr == notes.ErrNotFound
	components := make([]templ.Component, 0, 2)
	if useHX && !hasConflict {
		components = append(components, ui.NotesAutosaveResponse(mainProps))
	} else if useHX && hasConflict {
		components = append(components, ui.NotesMain(mainProps, true))
	} else {
		components = append(components, ui.NotesMain(mainProps, false))
	}
	if saveErr == notes.ErrConflict {
		components = append(components, ui.Alert(ui.PropsAlert{
			Label: "Sync conflict detected. You are viewing the latest server version.",
			Type:  ui.AlertTypeWarning,
		}))
	}
	if saveErr == notes.ErrNotFound {
		components = append(components, ui.Alert(ui.PropsAlert{
			Label: "That note no longer exists. Pick another or create a new one.",
			Type:  ui.AlertTypeInfo,
		}))
	}

	stts, err := render(w, r, components...)
	if err != nil {
		errorHTTP(w, stts, err)
	}
}

func NotesDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	store, err := getNotesStore()
	if err != nil {
		errorHTTP(w, http.StatusInternalServerError, err)
		return
	}

	if err := r.ParseForm(); err != nil {
		errorHTTP(w, http.StatusBadRequest, err)
		return
	}

	id := strings.TrimSpace(r.FormValue("id"))
	revision, _ := strconv.Atoi(r.FormValue("revision"))

	_, err = store.Delete(id, revision)
	if err != nil && err != notes.ErrConflict && err != notes.ErrNotFound {
		errorHTTP(w, http.StatusInternalServerError, err)
		return
	}
	if err == notes.ErrConflict {
		w.WriteHeader(http.StatusConflict)
	}
	if err == notes.ErrNotFound {
		w.WriteHeader(http.StatusNotFound)
	}

	mainProps, err := buildNotesMainProps("", false)
	if err != nil {
		errorHTTP(w, http.StatusInternalServerError, err)
		return
	}

	components := []templ.Component{ui.NotesMain(mainProps, false)}
	if err == notes.ErrConflict {
		components = append(components, ui.Alert(ui.PropsAlert{
			Label: "Note changed elsewhere. Showing the latest list.",
			Type:  ui.AlertTypeWarning,
		}))
	}
	if err == notes.ErrNotFound {
		components = append(components, ui.Alert(ui.PropsAlert{
			Label: "That note already disappeared. Sync refreshed the list.",
			Type:  ui.AlertTypeInfo,
		}))
	}

	stts, err := render(w, r, components...)
	if err != nil {
		errorHTTP(w, stts, err)
	}
}

func buildNotesMainProps(selectedID string, forceNew bool) (ui.PropsNotesMain, error) {
	store, err := getNotesStore()
	if err != nil {
		return ui.PropsNotesMain{}, err
	}

	items := store.List(false)
	var selected notes.Note
	hasSelected := false

	if !forceNew {
		if selectedID != "" {
			if note, ok := store.Get(selectedID); ok && !note.Deleted {
				selected = note
				hasSelected = true
			}
		}

		if !hasSelected && len(items) > 0 {
			selected = items[0]
			hasSelected = true
		}
	}

	listItems := make([]ui.NoteListItem, 0, len(items))
	for _, item := range items {
		listItems = append(listItems, ui.NoteListItem{
			ID:           item.ID,
			Title:        item.Title,
			UpdatedLabel: humanizeUpdated(item.UpdatedAt),
			UpdatedAt:    formatUpdatedAt(item.UpdatedAt),
			Selected:     hasSelected && item.ID == selected.ID,
		})
	}

	editor := ui.NoteEditor{}
	if hasSelected {
		editor = ui.NoteEditor{
			ID:           selected.ID,
			Title:        selected.Title,
			Body:         selected.Body,
			Revision:     selected.Revision,
			UpdatedLabel: humanizeUpdated(selected.UpdatedAt),
			UpdatedAt:    formatUpdatedAt(selected.UpdatedAt),
		}
	}

	return ui.PropsNotesMain{
		Items:        listItems,
		Editor:       editor,
		HasSelection: hasSelected,
	}, nil
}

func humanizeUpdated(updated time.Time) string {
	if updated.IsZero() {
		return "never"
	}

	diff := time.Since(updated)
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return strconv.Itoa(int(diff.Minutes())) + "m ago"
	case diff < 24*time.Hour:
		return strconv.Itoa(int(diff.Hours())) + "h ago"
	case diff < 7*24*time.Hour:
		return strconv.Itoa(int(diff.Hours()/24)) + "d ago"
	default:
		return updated.Format("Jan 2, 2006")
	}
}

func formatUpdatedAt(updated time.Time) string {
	if updated.IsZero() {
		return ""
	}
	return updated.Format(time.RFC3339)
}
