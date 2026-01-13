package notes

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound  = errors.New("note not found")
	ErrConflict  = errors.New("note revision conflict")
	ErrInvalidID = errors.New("invalid note id")
)

type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Revision  int       `json:"revision"`
	Deleted   bool      `json:"deleted"`
}

type Store struct {
	dir       string
	mu        sync.Mutex
	notes     map[string]Note
	filenames map[string]string
}

const notesMetadataFilename = "notes-metadata.json"
const notesFileExtension = ".md"

func NewStore(dir string) (*Store, error) {
	store := &Store{
		dir:       dir,
		notes:     make(map[string]Note),
		filenames: make(map[string]string),
	}

	if err := store.load(); err != nil {
		return nil, err
	}
	if _, err := store.SyncFromFilesystem(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *Store) List(includeDeleted bool) []Note {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]Note, 0, len(s.notes))
	for _, note := range s.notes {
		if note.Deleted && !includeDeleted {
			continue
		}
		items = append(items, note)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})

	return items
}

func (s *Store) Get(id string) (Note, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	note, ok := s.notes[id]
	return note, ok
}

func (s *Store) Create(folder, title, body string) (Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	folder = NormalizeFolder(folder)
	filename := s.uniqueFilename(slugify(title), folder)
	id := strings.TrimSuffix(filename, notesFileExtension)
	if !isValidID(id) {
		return Note{}, ErrInvalidID
	}
	title = NormalizeTitle(title)
	note := Note{
		ID:        id,
		Title:     title,
		Body:      body,
		CreatedAt: now,
		UpdatedAt: now,
		Revision:  1,
		Deleted:   false,
	}

	s.notes[id] = note
	s.filenames[id] = filename
	return note, s.save(note)
}

func (s *Store) Update(id, title, body string, expectedRevision int) (Note, error) {
	if id == "" {
		return Note{}, ErrInvalidID
	}
	if !isValidID(id) {
		return Note{}, ErrInvalidID
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	note, ok := s.notes[id]
	if !ok {
		return Note{}, ErrNotFound
	}

	if expectedRevision > 0 && expectedRevision != note.Revision {
		return note, ErrConflict
	}

	now := time.Now().UTC()
	note.Title = NormalizeTitle(title)
	note.Body = body
	note.UpdatedAt = now
	note.Revision++
	note.Deleted = false

	s.notes[id] = note
	return note, s.save(note)
}

func (s *Store) Delete(id string, expectedRevision int) (Note, error) {
	if id == "" {
		return Note{}, ErrInvalidID
	}
	if !isValidID(id) {
		return Note{}, ErrInvalidID
	}

	s.mu.Lock()
	note, ok := s.notes[id]
	if !ok {
		s.mu.Unlock()
		return Note{}, ErrNotFound
	}

	if expectedRevision > 0 && expectedRevision != note.Revision {
		s.mu.Unlock()
		return note, ErrConflict
	}

	filename := s.filenameFromMetadata(note.ID, s.filenames[note.ID])
	s.mu.Unlock()

	if filename == "" {
		return note, ErrNotFound
	}
	if err := os.Remove(s.notePath(filename)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return note, ErrNotFound
		}
		return note, err
	}

	_, err := s.SyncFromFilesystem()
	return note, err
}

func (s *Store) Move(id, folder string) (Note, string, error) {
	if id == "" {
		return Note{}, "", ErrInvalidID
	}
	if !isValidID(id) {
		return Note{}, "", ErrInvalidID
	}

	folder = NormalizeFolder(folder)

	s.mu.Lock()
	note, ok := s.notes[id]
	if !ok {
		s.mu.Unlock()
		return Note{}, "", ErrNotFound
	}
	filename := s.filenameFromMetadata(note.ID, s.filenames[note.ID])
	s.mu.Unlock()

	if filename == "" {
		return note, "", ErrNotFound
	}

	base := path.Base(filename)
	newFilename := base
	if folder != "" {
		newFilename = path.Join(folder, base)
	}
	if newFilename == filename {
		return note, id, nil
	}

	newFilename = s.uniqueFilenameOnDisk(newFilename, folder)
	oldPath := s.notePath(filename)
	newPath := s.notePath(newFilename)
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return note, "", err
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return note, "", err
	}

	_, err := s.SyncFromFilesystem()
	if err != nil {
		return note, "", err
	}

	newID := strings.TrimSuffix(newFilename, notesFileExtension)
	updated, ok := s.Get(newID)
	if ok {
		return updated, newID, nil
	}
	return note, newID, nil
}

func (s *Store) SyncFromFilesystem() (bool, error) {
	if _, err := os.Stat(s.dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	type fileSnapshot struct {
		filename string
		body     string
		modTime  time.Time
	}

	snapshots := make([]fileSnapshot, 0)
	err := filepath.WalkDir(s.dir, func(entryPath string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, os.ErrNotExist) {
				return nil
			}
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(d.Name()) != notesFileExtension {
			return nil
		}
		if d.Name() == notesMetadataFilename {
			return nil
		}
		rel, err := filepath.Rel(s.dir, entryPath)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !isValidID(strings.TrimSuffix(rel, notesFileExtension)) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		body, err := s.loadBodyByFilename(rel)
		if err != nil {
			return err
		}
		snapshots = append(snapshots, fileSnapshot{
			filename: rel,
			body:     body,
			modTime:  info.ModTime().UTC(),
		})
		return nil
	})
	if err != nil {
		return false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	nextNotes := make(map[string]Note, len(snapshots))
	nextFilenames := make(map[string]string, len(snapshots))
	changed := len(s.notes) != len(snapshots)

	for _, snapshot := range snapshots {
		id := strings.TrimSuffix(snapshot.filename, notesFileExtension)
		if !isValidID(id) {
			continue
		}
		updatedAt := snapshot.modTime
		if updatedAt.IsZero() {
			updatedAt = time.Now().UTC()
		}
		title := titleFromFile(snapshot.body, snapshot.filename)
		existing, ok := s.notes[id]
		revision := 1
		createdAt := updatedAt
		if ok {
			createdAt = existing.CreatedAt
			revision = existing.Revision
			if existing.Body != snapshot.body {
				revision++
			}
			if existing.Deleted {
				changed = true
			}
		} else {
			changed = true
		}

		note := Note{
			ID:        id,
			Title:     title,
			Body:      snapshot.body,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			Revision:  revision,
			Deleted:   false,
		}
		nextNotes[id] = note
		nextFilenames[id] = snapshot.filename

		if ok {
			if existing.Title != note.Title || existing.Body != note.Body || !existing.UpdatedAt.Equal(note.UpdatedAt) {
				changed = true
			}
		}
	}

	if !changed {
		return false, nil
	}

	s.notes = nextNotes
	s.filenames = nextFilenames

	if err := s.saveMetadata(); err != nil {
		return false, err
	}

	return true, nil
}

func (s *Store) load() error {
	if s.dir == "" {
		return errors.New("notes store directory cannot be empty")
	}

	metaPath := filepath.Join(s.dir, notesMetadataFilename)
	meta, err := s.loadMetadata(metaPath)
	if err != nil {
		return err
	}
	hasMetadata := len(meta) > 0

	for id, info := range meta {
		if !isValidID(id) {
			continue
		}
		filename := s.filenameFromMetadata(id, info.Filename)
		body, err := s.loadBodyByFilename(filename)
		if err != nil {
			return err
		}
		s.filenames[id] = filename
		title := NormalizeTitle(info.Title)
		if title == defaultTitle && strings.TrimSpace(info.Title) == "" {
			title = NormalizeTitleFromBody(body)
		}
		s.notes[id] = Note{
			ID:        id,
			Title:     title,
			Body:      body,
			CreatedAt: info.CreatedAt,
			UpdatedAt: info.UpdatedAt,
			Revision:  info.Revision,
			Deleted:   info.Deleted,
		}
	}

	if !hasMetadata {
		entries, err := os.ReadDir(s.dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if filepath.Ext(entry.Name()) != notesFileExtension {
				continue
			}
			id := strings.TrimSuffix(entry.Name(), notesFileExtension)
			if !isValidID(id) {
				continue
			}
			if _, exists := s.notes[id]; exists {
				continue
			}
			body, err := s.loadBodyByFilename(entry.Name())
			if err != nil {
				return err
			}
			now := time.Now().UTC()
			s.filenames[id] = entry.Name()
			s.notes[id] = Note{
				ID:        id,
				Title:     NormalizeTitleFromBody(body),
				Body:      body,
				CreatedAt: now,
				UpdatedAt: now,
				Revision:  1,
				Deleted:   false,
			}
		}
	}

	return nil
}

func (s *Store) save(note Note) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}

	if err := s.saveBody(note); err != nil {
		return err
	}

	return s.saveMetadata()
}

func (s *Store) saveMetadata() error {
	payload, err := json.MarshalIndent(s.metadataSnapshot(), "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(s.dir, notesMetadataFilename)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, path)
}

type noteMetadata struct {
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Revision  int       `json:"revision"`
	Deleted   bool      `json:"deleted"`
	Filename  string    `json:"filename"`
}

func (s *Store) metadataSnapshot() map[string]noteMetadata {
	meta := make(map[string]noteMetadata, len(s.notes))
	for id, note := range s.notes {
		filename := s.filenameFromMetadata(id, s.filenames[id])
		meta[id] = noteMetadata{
			Title:     note.Title,
			CreatedAt: note.CreatedAt,
			UpdatedAt: note.UpdatedAt,
			Revision:  note.Revision,
			Deleted:   note.Deleted,
			Filename:  filename,
		}
	}
	return meta
}

func (s *Store) loadMetadata(path string) (map[string]noteMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]noteMetadata{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return map[string]noteMetadata{}, nil
	}

	var meta map[string]noteMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	if meta == nil {
		return map[string]noteMetadata{}, nil
	}
	return meta, nil
}

func (s *Store) loadBodyByFilename(filename string) (string, error) {
	path := s.notePath(filename)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func (s *Store) saveBody(note Note) error {
	filename := s.ensureFilename(note)
	path := s.notePath(filename)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(note.Body), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) notePath(filename string) string {
	return filepath.Join(s.dir, filepath.FromSlash(filename))
}

func (s *Store) filenameFromMetadata(id, filename string) string {
	if filename == "" {
		return id + notesFileExtension
	}
	if filepath.Ext(filename) == "" {
		return filename + notesFileExtension
	}
	return filename
}

func (s *Store) ensureFilename(note Note) string {
	current := s.filenameFromMetadata(note.ID, s.filenames[note.ID])
	if current != "" {
		s.filenames[note.ID] = current
		return current
	}
	filename := s.filenameFromMetadata(note.ID, note.ID)
	s.filenames[note.ID] = filename
	return filename
}

func titleFromFile(body, filename string) string {
	title := NormalizeTitleFromBody(body)
	if title != defaultTitle {
		return title
	}
	base := path.Base(strings.TrimSuffix(filename, notesFileExtension))
	if base == "" {
		return title
	}
	return NormalizeTitle(base)
}

func (s *Store) uniqueFilename(base, folder string) string {
	if base == "" {
		base = "note"
	}
	folder = NormalizeFolder(folder)
	candidate := path.Join(folder, base) + notesFileExtension
	if s.isFilenameAvailable(candidate, "") {
		return candidate
	}
	for i := 2; ; i++ {
		candidate = path.Join(folder, fmt.Sprintf("%s-%d%s", base, i, notesFileExtension))
		if s.isFilenameAvailable(candidate, "") {
			return candidate
		}
	}
}

func (s *Store) uniqueFilenameOnDisk(filename, folder string) string {
	if _, err := os.Stat(s.notePath(filename)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return filename
		}
	}

	base := path.Base(strings.TrimSuffix(filename, notesFileExtension))
	if base == "" {
		base = "note"
	}
	folder = NormalizeFolder(folder)
	for i := 2; ; i++ {
		candidate := path.Join(folder, fmt.Sprintf("%s-%d%s", base, i, notesFileExtension))
		if _, err := os.Stat(s.notePath(candidate)); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return candidate
			}
		}
	}
}

func (s *Store) isFilenameAvailable(filename, id string) bool {
	for existingID, existing := range s.filenames {
		if existing == filename && existingID != id {
			return false
		}
	}
	return true
}

func isValidID(id string) bool {
	if id == "" {
		return false
	}
	if strings.Contains(id, "\\") {
		return false
	}
	if strings.HasPrefix(id, "/") {
		return false
	}
	clean := path.Clean(id)
	if clean != id {
		return false
	}
	parts := strings.Split(id, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func NormalizeFolder(folder string) string {
	folder = strings.TrimSpace(folder)
	folder = strings.Trim(folder, "/")
	if folder == "" || folder == "." {
		return ""
	}
	if !isValidID(folder) {
		return ""
	}
	return folder
}

func slugify(value string) string {
	lower := strings.ToLower(value)
	var b strings.Builder
	b.Grow(len(lower))
	previousDash := false
	for _, r := range lower {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			previousDash = false
			continue
		}
		if previousDash {
			continue
		}
		b.WriteByte('-')
		previousDash = true
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "note"
	}
	if len(slug) > 80 {
		return strings.Trim(slug[:80], "-")
	}
	return slug
}
