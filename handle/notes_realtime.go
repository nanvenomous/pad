package handle

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nanvenomous/pad/notes"
	"github.com/nanvenomous/pad/ui"
	"golang.org/x/net/websocket"
)

type notesStreamClient struct {
	ch         chan []byte
	selectedID string
	forceNew   bool
	closeOnce  sync.Once
}

type notesHub struct {
	mu      sync.Mutex
	clients map[*notesStreamClient]struct{}
}

var (
	notesRealtimeOnce sync.Once
	notesHubInstance  *notesHub
)

func initNotesRealtime(store *notes.Store, dir string) {
	notesRealtimeOnce.Do(func() {
		notesHubInstance = &notesHub{
			clients: make(map[*notesStreamClient]struct{}),
		}
		startNotesWatcher(store, dir)
	})
}

func (h *notesHub) register(client *notesStreamClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = struct{}{}
}

func (h *notesHub) unregister(client *notesStreamClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[client]; !ok {
		return
	}
	delete(h.clients, client)
	client.closeOnce.Do(func() {
		close(client.ch)
	})
}

func (h *notesHub) snapshot() []*notesStreamClient {
	h.mu.Lock()
	defer h.mu.Unlock()
	clients := make([]*notesStreamClient, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	return clients
}

func broadcastNotesUpdate() {
	if notesHubInstance == nil || notesStore == nil {
		return
	}
	notesHubInstance.broadcast(notesStore)
}

func (h *notesHub) broadcast(store *notes.Store) {
	clients := h.snapshot()
	for _, client := range clients {
		payload, err := renderNotesUpdate(client.selectedID, client.forceNew)
		if err != nil {
			log.Printf("notes stream render: %v", err)
			continue
		}
		h.safeSend(client, payload)
	}
}

func (h *notesHub) safeSend(client *notesStreamClient, payload []byte) {
	defer func() {
		if recover() != nil {
			h.unregister(client)
		}
	}()

	select {
	case client.ch <- payload:
	default:
	}
}

func renderNotesUpdate(selectedID string, forceNew bool) ([]byte, error) {
	props, err := buildNotesMainProps(selectedID, forceNew, "")
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := ui.NotesMain(props, true).Render(context.Background(), &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func NotesStreamHandler(w http.ResponseWriter, r *http.Request) {
	store, err := getNotesStore()
	if err != nil {
		errorHTTP(w, http.StatusInternalServerError, err)
		return
	}

	selectedID := strings.TrimSpace(r.URL.Query().Get("id"))
	if selectedID != "" {
		if _, ok := store.Get(selectedID); !ok {
			selectedID = ""
		}
	}

	forceNew, _ := strconv.ParseBool(r.URL.Query().Get("new"))

	if notesHubInstance == nil {
		http.Error(w, "Notes stream unavailable", http.StatusServiceUnavailable)
		return
	}

	websocket.Handler(func(ws *websocket.Conn) {
		client := &notesStreamClient{
			ch:         make(chan []byte, 8),
			selectedID: selectedID,
			forceNew:   forceNew,
		}
		notesHubInstance.register(client)
		defer notesHubInstance.unregister(client)

		done := make(chan struct{})
		go func() {
			for msg := range client.ch {
				if err := websocket.Message.Send(ws, string(msg)); err != nil {
					break
				}
			}
			close(done)
		}()

		for {
			var payload string
			if err := websocket.Message.Receive(ws, &payload); err != nil {
				break
			}
		}

		<-done
	}).ServeHTTP(w, r)
}

func startNotesWatcher(store *notes.Store, dir string) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("notes watcher mkdir: %v", err)
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("notes watcher init: %v", err)
		return
	}
	watched := make(map[string]struct{})

	addWatch := func(path string) {
		if _, ok := watched[path]; ok {
			return
		}
		if err := watcher.Add(path); err != nil {
			log.Printf("notes watcher add: %v", err)
			return
		}
		watched[path] = struct{}{}
	}

	err = filepath.WalkDir(dir, func(entryPath string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if d.IsDir() {
			addWatch(entryPath)
		}
		return nil
	})
	if err != nil {
		_ = watcher.Close()
		log.Printf("notes watcher add: %v", err)
		return
	}

	go func() {
		defer watcher.Close()
		timer := time.NewTimer(0)
		if !timer.Stop() {
			<-timer.C
		}
		pending := false
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
					continue
				}
				if event.Op&fsnotify.Create != 0 {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						addWatch(event.Name)
					}
				}
				pending = true
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(200 * time.Millisecond)
			case <-timer.C:
				if !pending {
					continue
				}
				pending = false
				changed, err := store.SyncFromFilesystem()
				if err != nil {
					log.Printf("notes watcher sync: %v", err)
					continue
				}
				if changed {
					broadcastNotesUpdate()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("notes watcher error: %v", err)
			}
		}
	}()
}
