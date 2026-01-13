package handle

import (
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
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
		if notesStoreErr == nil {
			initNotesRealtime(notesStore, dir)
		}
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
		mux.HandleFunc("/notes/move-modal", NotesMoveModalHandler)
		mux.HandleFunc("/notes/move", NotesMoveHandler)
		mux.HandleFunc("/notes/stream", NotesStreamHandler)
	})
}

func NotesPageHandler(w http.ResponseWriter, r *http.Request) {
	mainProps, err := buildNotesMainProps(r.URL.Query().Get("id"), false, "")
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
	mainProps, err := buildNotesMainProps(r.URL.Query().Get("id"), false, "")
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
	mainProps, err := buildNotesMainProps("", true, r.URL.Query().Get("folder"))
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

	time.Sleep(150 * time.Millisecond)

	id := strings.TrimSpace(r.FormValue("id"))
	body := r.FormValue("body")
	revision, _ := strconv.Atoi(r.FormValue("revision"))
	folder := strings.TrimSpace(r.FormValue("folder"))

	title := notes.NormalizeTitleFromBody(body)

	didSave := false
	var note notes.Note
	if id == "" {
		note, err = store.Create(folder, title, body)
		if err != nil {
			errorHTTP(w, http.StatusInternalServerError, err)
			return
		}
		didSave = true
	} else {
		note, err = store.Update(id, title, body, revision)
		if err != nil && err != notes.ErrConflict && err != notes.ErrNotFound {
			errorHTTP(w, http.StatusInternalServerError, err)
			return
		}
		if err == nil {
			didSave = true
		}
		if err == notes.ErrConflict {
			w.WriteHeader(http.StatusConflict)
		}
		if err == notes.ErrNotFound {
			w.WriteHeader(http.StatusNotFound)
		}
	}
	if didSave {
		broadcastNotesUpdate()
	}

	saveErr := err
	selectedID := note.ID
	if saveErr == notes.ErrNotFound {
		selectedID = ""
	}
	if saveErr == nil && note.ID != "" {
		w.Header().Set("HX-Push-Url", "/?id="+url.QueryEscape(note.ID))
	}

	mainProps, err := buildNotesMainProps(selectedID, false, "")
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
	if err == nil {
		broadcastNotesUpdate()
	}
	if err == notes.ErrConflict {
		w.WriteHeader(http.StatusConflict)
	}
	if err == notes.ErrNotFound {
		w.WriteHeader(http.StatusNotFound)
	}

	mainProps, err := buildNotesMainProps("", false, "")
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
	components = append(components,
		ui.EmptyModalPopover(ui.PropsEmptyModalPopover{}),
	)

	stts, err := render(w, r, components...)
	if err != nil {
		errorHTTP(w, stts, err)
	}
}

func NotesMoveModalHandler(w http.ResponseWriter, r *http.Request) {
	store, err := getNotesStore()
	if err != nil {
		errorHTTP(w, http.StatusInternalServerError, err)
		return
	}

	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		errorHTTP(w, http.StatusBadRequest, notes.ErrInvalidID)
		return
	}

	note, ok := store.Get(id)
	if !ok || note.Deleted {
		errorHTTP(w, http.StatusNotFound, notes.ErrNotFound)
		return
	}

	folders := collectFolders(store.List(false))
	props := ui.PropsNoteActions{
		ID:       note.ID,
		Title:    note.Title,
		Folder:   folderFromID(note.ID),
		Revision: note.Revision,
		Folders:  folders,
	}

	stts, err := render(w, r, ui.NoteMoveModal(props))
	if err != nil {
		errorHTTP(w, stts, err)
	}
}

func NotesMoveHandler(w http.ResponseWriter, r *http.Request) {
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
	folder := strings.TrimSpace(r.FormValue("folder"))
	if id == "" {
		errorHTTP(w, http.StatusBadRequest, notes.ErrInvalidID)
		return
	}

	note, newID, err := store.Move(id, folder)
	if err != nil && err != notes.ErrNotFound {
		errorHTTP(w, http.StatusInternalServerError, err)
		return
	}
	if err == nil {
		broadcastNotesUpdate()
	}

	selectedID := newID
	if selectedID == "" {
		selectedID = note.ID
	}

	mainProps, err := buildNotesMainProps(selectedID, false, "")
	if err != nil {
		errorHTTP(w, http.StatusInternalServerError, err)
		return
	}

	components := []templ.Component{
		ui.NotesMain(mainProps, false),
		ui.EmptyModalPopover(ui.PropsEmptyModalPopover{}),
	}

	stts, err := render(w, r, components...)
	if err != nil {
		errorHTTP(w, stts, err)
	}
}

func buildNotesMainProps(selectedID string, forceNew bool, newFolder string) (ui.PropsNotesMain, error) {
	store, err := getNotesStore()
	if err != nil {
		return ui.PropsNotesMain{}, err
	}

	items := store.List(false)
	state := buildNotesListState(store, items, selectedID, forceNew, newFolder)
	folders := buildFolderTree(state.listItems)

	return ui.PropsNotesMain{
		Folders:       folders,
		Items:         state.listItems,
		Editor:        state.editor,
		HasSelection:  state.hasSelected,
		ForceNew:      forceNew,
		NotesCount:    len(state.listItems),
		CurrentFolder: state.currentFolder,
	}, nil
}

type notesListState struct {
	listItems     []ui.NoteListItem
	editor        ui.NoteEditor
	hasSelected   bool
	currentFolder string
}

func buildNotesListState(store *notes.Store, items []notes.Note, selectedID string, forceNew bool, newFolder string) notesListState {
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
		folder := folderFromID(item.ID)
		listItems = append(listItems, ui.NoteListItem{
			ID:           item.ID,
			Title:        item.Title,
			UpdatedLabel: "",
			UpdatedAt:    formatUpdatedAt(item.UpdatedAt),
			Selected:     hasSelected && item.ID == selected.ID,
			Folder:       folder,
			Revision:     item.Revision,
		})
	}

	editor := ui.NoteEditor{}
	currentFolder := notes.NormalizeFolder(newFolder)
	if hasSelected {
		currentFolder = folderFromID(selected.ID)
		editor = ui.NoteEditor{
			ID:           selected.ID,
			Title:        selected.Title,
			Body:         selected.Body,
			Revision:     selected.Revision,
			UpdatedLabel: "",
			UpdatedAt:    formatUpdatedAt(selected.UpdatedAt),
			Folder:       currentFolder,
		}
	}

	return notesListState{
		listItems:     listItems,
		editor:        editor,
		hasSelected:   hasSelected,
		currentFolder: currentFolder,
	}
}

type folderNode struct {
	name     string
	path     string
	notes    []ui.NoteListItem
	children []*folderNode
}

func buildFolderTree(items []ui.NoteListItem) []ui.NoteFolder {
	rootNotes := make([]ui.NoteListItem, 0)
	nodes := make(map[string]*folderNode)

	ensureNode := func(folderPath string) *folderNode {
		if node, ok := nodes[folderPath]; ok {
			return node
		}
		node := &folderNode{
			name: path.Base(folderPath),
			path: folderPath,
		}
		nodes[folderPath] = node
		return node
	}

	for _, item := range items {
		if item.Folder == "" {
			rootNotes = append(rootNotes, item)
			continue
		}
		segments := strings.Split(item.Folder, "/")
		for i := range segments {
			if segments[i] == "" {
				continue
			}
			segmentPath := strings.Join(segments[:i+1], "/")
			ensureNode(segmentPath)
		}
		node := ensureNode(item.Folder)
		node.notes = append(node.notes, item)
	}

	paths := make([]string, 0, len(nodes))
	for folderPath := range nodes {
		paths = append(paths, folderPath)
	}
	sort.Slice(paths, func(i, j int) bool {
		return strings.Count(paths[i], "/") < strings.Count(paths[j], "/")
	})

	top := make([]*folderNode, 0)
	for _, folderPath := range paths {
		node := nodes[folderPath]
		parent := path.Dir(folderPath)
		if parent == "." || parent == "" {
			top = append(top, node)
			continue
		}
		parentNode, ok := nodes[parent]
		if !ok {
			top = append(top, node)
			continue
		}
		parentNode.children = append(parentNode.children, node)
	}

	sortFolderNodes(top)

	folders := make([]ui.NoteFolder, 0)
	if len(rootNotes) > 0 {
		root := &folderNode{
			name:  "Root",
			path:  "",
			notes: rootNotes,
		}
		folders = append(folders, convertFolderNode(root))
	}
	for _, node := range top {
		folders = append(folders, convertFolderNode(node))
	}
	return folders
}

func sortFolderNodes(nodes []*folderNode) {
	for _, node := range nodes {
		sortFolderNodes(node.children)
		sort.Slice(node.children, func(i, j int) bool {
			return node.children[i].name < node.children[j].name
		})
		sort.Slice(node.notes, func(i, j int) bool {
			return node.notes[i].UpdatedAt > node.notes[j].UpdatedAt
		})
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].name < nodes[j].name
	})
}

func convertFolderNode(node *folderNode) ui.NoteFolder {
	children := make([]ui.NoteFolder, 0, len(node.children))
	for _, child := range node.children {
		children = append(children, convertFolderNode(child))
	}
	count := len(node.notes)
	for i := range children {
		count += children[i].Count
	}
	return ui.NoteFolder{
		Name:     node.name,
		Path:     node.path,
		Open:     true,
		Notes:    node.notes,
		Children: children,
		Count:    count,
	}
}

func folderFromID(id string) string {
	dir := path.Dir(id)
	if dir == "." {
		return ""
	}
	return dir
}

func collectFolders(items []notes.Note) []string {
	folders := make(map[string]struct{})
	folders[""] = struct{}{}
	for _, item := range items {
		dir := folderFromID(item.ID)
		folders[dir] = struct{}{}
	}
	values := make([]string, 0, len(folders))
	for folder := range folders {
		values = append(values, folder)
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i] < values[j]
	})
	return values
}

func formatUpdatedAt(updated time.Time) string {
	if updated.IsZero() {
		return ""
	}
	return updated.Format(time.RFC3339)
}
