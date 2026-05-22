package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"bigtable/internal/telemetry"
)

type Engine interface {
	Put(key, value []byte) error
	Delete(key []byte) error
	Get(key []byte) ([]byte, bool, error)
	Compact() error
	Snapshot() telemetry.Snapshot
	Events() <-chan telemetry.Event
}

type Server struct {
	engine Engine
	mux    *http.ServeMux

	subsMu sync.RWMutex
	subs   map[chan telemetry.Event]struct{}
}

type commandRequest struct {
	Op    string `json:"op"`
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

func NewServer(engine Engine) *Server {
	s := &Server{
		engine: engine,
		mux:    http.NewServeMux(),
		subs:   make(map[chan telemetry.Event]struct{}),
	}
	s.routes()
	go s.broadcast()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/state", s.handleState)
	s.mux.HandleFunc("/api/command", s.handleCommand)
	s.mux.HandleFunc("/api/events", s.handleEvents)
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.engine.Snapshot())
}

func (s *Server) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req commandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var err error
	switch req.Op {
	case "put":
		err = s.engine.Put([]byte(req.Key), []byte(req.Value))
	case "get":
		_, _, err = s.engine.Get([]byte(req.Key))
	case "delete":
		err = s.engine.Delete([]byte(req.Key))
	case "compact":
		err = s.engine.Compact()
	default:
		http.Error(w, "unknown op", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	client := make(chan telemetry.Event, 64)

	s.subsMu.Lock()
	s.subs[client] = struct{}{}
	s.subsMu.Unlock()

	defer func() {
		s.subsMu.Lock()
		delete(s.subs, client)
		close(client)
		s.subsMu.Unlock()
	}()

	notify := r.Context().Done()

	for {
		select {
		case ev := <-client:
			if err := writeSSE(w, ev); err != nil {
				return
			}
			flusher.Flush()
		case <-notify:
			return
		}
	}
}

func (s *Server) broadcast() {
	for ev := range s.engine.Events() {
		s.subsMu.RLock()
		for ch := range s.subs {
			select {
			case ch <- ev:
			default:
			}
		}
		s.subsMu.RUnlock()
	}
}

func writeSSE(w http.ResponseWriter, ev telemetry.Event) error {
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	return err
}