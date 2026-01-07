package notes

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
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
	path  string
	mu    sync.Mutex
	notes map[string]Note
}

func NewStore(path string) (*Store, error) {
	store := &Store{
		path:  path,
		notes: make(map[string]Note),
	}

	if err := store.load(); err != nil {
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

func (s *Store) Since(since time.Time, includeDeleted bool) []Note {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]Note, 0)
	for _, note := range s.notes {
		if note.UpdatedAt.Before(since) {
			continue
		}
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

func (s *Store) Create(title, body string) (Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, err := gonanoid.New()
	if err != nil {
		return Note{}, err
	}

	now := time.Now().UTC()
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
	return note, s.save()
}

func (s *Store) Update(id, title, body string, expectedRevision int) (Note, error) {
	if id == "" {
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
	note.Title = title
	note.Body = body
	note.UpdatedAt = now
	note.Revision++
	note.Deleted = false

	s.notes[id] = note
	return note, s.save()
}

func (s *Store) Upsert(id, title, body string, expectedRevision int, allowCreate bool) (Note, error) {
	if id == "" {
		return Note{}, ErrInvalidID
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	note, ok := s.notes[id]
	if !ok {
		if !allowCreate {
			return Note{}, ErrNotFound
		}

		now := time.Now().UTC()
		revision := 1
		if expectedRevision > 0 {
			revision = expectedRevision
		}

		note = Note{
			ID:        id,
			Title:     title,
			Body:      body,
			CreatedAt: now,
			UpdatedAt: now,
			Revision:  revision,
			Deleted:   false,
		}
		s.notes[id] = note
		return note, s.save()
	}

	if expectedRevision > 0 && expectedRevision != note.Revision {
		return note, ErrConflict
	}

	note.Title = title
	note.Body = body
	note.UpdatedAt = time.Now().UTC()
	note.Revision++
	note.Deleted = false
	s.notes[id] = note

	return note, s.save()
}

func (s *Store) Delete(id string, expectedRevision int) (Note, error) {
	if id == "" {
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

	note.Deleted = true
	note.UpdatedAt = time.Now().UTC()
	note.Revision++
	s.notes[id] = note

	return note, s.save()
}

func (s *Store) load() error {
	if s.path == "" {
		return errors.New("notes store path cannot be empty")
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	if len(data) == 0 {
		return nil
	}

	return json.Unmarshal(data, &s.notes)
}

func (s *Store) save() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(s.notes, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, s.path)
}
