// Package web serves a page for managing agent sessions: list, create and delete them, chat in one, watch the agent
// loop live and answer its permission requests.
package web

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	agentloop "github.com/liuliqiang/llmagent/99_tag_iterate_version"
	"github.com/liuliqiang/log4go"
)

//go:embed index.html
var indexHTML []byte

// keepAliveInterval is how often an idle event stream sends a comment, so proxies don't drop it.
var keepAliveInterval = 20 * time.Second

// NewHandler routes the page and its API:
//
//	GET    /                                   the page
//	GET    /api/sessions                       list sessions
//	POST   /api/sessions                       create one, optionally starting it with {"prompt"}
//	GET    /api/sessions/{id}                  one session with its turns
//	DELETE /api/sessions/{id}                  delete it
//	POST   /api/sessions/{id}/messages         send {"prompt"}; 409 while the session is busy
//	GET    /api/sessions/{id}/events           server-sent events: a snapshot, then every change
//	POST   /api/sessions/{id}/permissions/{pid} answer {"allow"}
func NewHandler(m *agentloop.SessionManager) http.Handler {
	s := &server{m: m}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.index)
	mux.HandleFunc("GET /api/sessions", s.list)
	mux.HandleFunc("POST /api/sessions", s.create)
	mux.HandleFunc("GET /api/sessions/{id}", s.get)
	mux.HandleFunc("DELETE /api/sessions/{id}", s.delete)
	mux.HandleFunc("POST /api/sessions/{id}/messages", s.send)
	mux.HandleFunc("GET /api/sessions/{id}/events", s.events)
	mux.HandleFunc("POST /api/sessions/{id}/permissions/{pid}", s.permission)
	return mux
}

type server struct {
	m *agentloop.SessionManager
}

type promptRequest struct {
	Prompt string `json:"prompt"`
}

func (s *server) index(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func (s *server) list(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.m.List())
}

func (s *server) create(w http.ResponseWriter, r *http.Request) {
	var req promptRequest
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log4go.DefaultLogger().Error(r.Context(), "decode create request failed: %v", err)
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	session, err := s.m.Create()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if req.Prompt != "" {
		if err := session.Submit(req.Prompt); err != nil {
			log4go.DefaultLogger().Error(r.Context(), "start new session failed: %v, session: %s", err, session.ID())
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	writeJSON(w, http.StatusCreated, session.View(false))
}

func (s *server) get(w http.ResponseWriter, r *http.Request) {
	session, ok := s.session(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, session.View(true))
}

func (s *server) delete(w http.ResponseWriter, r *http.Request) {
	if err := s.m.Delete(r.PathValue("id")); err != nil {
		writeError(w, statusOf(err), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) send(w http.ResponseWriter, r *http.Request) {
	session, ok := s.session(w, r)
	if !ok {
		return
	}
	var req promptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log4go.DefaultLogger().Error(r.Context(), "decode message request failed: %v, session: %s", err, session.ID())
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := session.Submit(req.Prompt); err != nil {
		writeError(w, statusOf(err), err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *server) permission(w http.ResponseWriter, r *http.Request) {
	session, ok := s.session(w, r)
	if !ok {
		return
	}
	var req struct {
		Allow bool `json:"allow"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log4go.DefaultLogger().Error(r.Context(), "decode permission answer failed: %v, session: %s", err, session.ID())
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := session.ResolvePermission(r.PathValue("pid"), req.Allow); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// events streams the session: a snapshot first, then every change. When the session drops a slow subscriber the
// stream ends and the browser's EventSource reconnects, starting again from a snapshot.
func (s *server) events(w http.ResponseWriter, r *http.Request) {
	session, ok := s.session(w, r)
	if !ok {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, errors.New("streaming unsupported"))
		return
	}
	snapshot, events, cancel := session.Subscribe()
	defer cancel()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	if err := writeEvent(w, agentloop.SessionEvent{Type: "snapshot", Session: &snapshot}); err != nil {
		return
	}
	flusher.Flush()

	keepAlive := time.NewTicker(keepAliveInterval)
	defer keepAlive.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-keepAlive.C:
			fmt.Fprint(w, ": keep-alive\n\n")
		case event, ok := <-events:
			if !ok {
				return
			}
			if err := writeEvent(w, event); err != nil {
				return
			}
		}
		flusher.Flush()
	}
}

func (s *server) session(w http.ResponseWriter, r *http.Request) (*agentloop.Session, bool) {
	session, err := s.m.Get(r.PathValue("id"))
	if err != nil {
		writeError(w, statusOf(err), err)
		return nil, false
	}
	return session, true
}

func writeEvent(w http.ResponseWriter, event agentloop.SessionEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		log4go.DefaultLogger().Error(context.Background(), "encode session event failed: %v, type: %s", err, event.Type)
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	return err
}

func statusOf(err error) int {
	switch {
	case errors.Is(err, agentloop.ErrSessionNotFound):
		return http.StatusNotFound
	case errors.Is(err, agentloop.ErrSessionBusy):
		return http.StatusConflict
	default:
		return http.StatusBadRequest
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
