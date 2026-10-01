package web

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	agentloop "github.com/liuliqiang/llmagent/99_tag_iterate_version"
)

// blockingLLM answers every call with reply, but only after release is closed.
type blockingLLM struct {
	reply   string
	release chan struct{}
	once    sync.Once
}

func (l *blockingLLM) GetModel() agentloop.Model { return "fake" }

func (l *blockingLLM) SendMessages(ctx context.Context, _ agentloop.Model, _ agentloop.Message, _ []agentloop.Message, _ []agentloop.Tool, _ agentloop.SendMessagesOpts) (agentloop.SendMessagesResponse, error) {
	select {
	case <-l.release:
	case <-ctx.Done():
		return agentloop.SendMessagesResponse{}, ctx.Err()
	}
	return agentloop.SendMessagesResponse{Content: []agentloop.MessagesBlock{{Type: agentloop.MessagesBlockTypeText, Text: l.reply}}}, nil
}

func (l *blockingLLM) unblock() { l.once.Do(func() { close(l.release) }) }

func newTestServer(t *testing.T) (*httptest.Server, *blockingLLM) {
	t.Helper()
	t.Chdir(t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	llm := &blockingLLM{reply: "all done", release: make(chan struct{})}
	t.Cleanup(func() {
		llm.unblock()
		cancel()
	})
	m, err := agentloop.NewSessionManager(ctx, filepath.Join(t.TempDir(), "sessions"), llm, new(agentloop.Hooks))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(NewHandler(m))
	t.Cleanup(srv.Close)
	return srv, llm
}

func do(t *testing.T, method, url, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func decode[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

// readEvents returns the events of an SSE stream as they arrive.
func readEvents(t *testing.T, body io.Reader) <-chan agentloop.SessionEvent {
	t.Helper()
	out := make(chan agentloop.SessionEvent, 64)
	go func() {
		defer close(out)
		scanner := bufio.NewScanner(body)
		scanner.Buffer(make([]byte, 1<<20), 1<<20)
		for scanner.Scan() {
			line, ok := strings.CutPrefix(scanner.Text(), "data: ")
			if !ok {
				continue
			}
			var ev agentloop.SessionEvent
			if err := json.Unmarshal([]byte(line), &ev); err != nil {
				t.Errorf("bad event %q: %v", line, err)
				return
			}
			out <- ev
		}
	}()
	return out
}

func waitFor(t *testing.T, events <-chan agentloop.SessionEvent, match func(agentloop.SessionEvent) bool) agentloop.SessionEvent {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				t.Fatal("event stream ended")
			}
			if match(ev) {
				return ev
			}
		case <-timeout:
			t.Fatal("timed out waiting for event")
		}
	}
}

func TestIndexServed(t *testing.T) {
	srv, _ := newTestServer(t)
	resp := do(t, http.MethodGet, srv.URL+"/", "")
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "Agent Sessions") {
		t.Errorf("status %d", resp.StatusCode)
	}
	if resp := do(t, http.MethodGet, srv.URL+"/nope", ""); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown path status %d", resp.StatusCode)
	}
}

func TestQuickStartStreamsTurnUntilReply(t *testing.T) {
	srv, llm := newTestServer(t)

	resp := do(t, http.MethodPost, srv.URL+"/api/sessions", `{"prompt":"fix the tests"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status %d", resp.StatusCode)
	}
	created := decode[agentloop.SessionView](t, resp)
	if created.Title != "fix the tests" || !created.Busy {
		t.Errorf("created = %+v", created)
	}

	stream := do(t, http.MethodGet, srv.URL+"/api/sessions/"+created.ID+"/events", "")
	if ct := stream.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type %q", ct)
	}
	events := readEvents(t, stream.Body)
	if first := <-events; first.Type != "snapshot" || first.Session == nil || first.Session.ID != created.ID {
		t.Fatalf("first event = %+v", first)
	}

	// busy while the model is thinking
	if resp := do(t, http.MethodPost, srv.URL+"/api/sessions/"+created.ID+"/messages", `{"prompt":"more"}`); resp.StatusCode != http.StatusConflict {
		t.Errorf("send while busy status %d", resp.StatusCode)
	}

	llm.unblock()
	done := waitFor(t, events, func(ev agentloop.SessionEvent) bool {
		return ev.Type == "turn" && ev.Turn.Status == "done"
	})
	if done.Turn.Reply != "all done" || done.Turn.Prompt != "fix the tests" {
		t.Errorf("finished turn = %+v", done.Turn)
	}
	waitFor(t, events, func(ev agentloop.SessionEvent) bool { return ev.Type == "session" && !ev.Session.Busy })

	if resp := do(t, http.MethodPost, srv.URL+"/api/sessions/"+created.ID+"/messages", `{"prompt":"and now?"}`); resp.StatusCode != http.StatusAccepted {
		t.Errorf("follow-up status %d", resp.StatusCode)
	}
	waitFor(t, events, func(ev agentloop.SessionEvent) bool {
		return ev.Type == "turn" && ev.Turn.ID == 2 && ev.Turn.Status == "done"
	})

	got := decode[agentloop.SessionView](t, do(t, http.MethodGet, srv.URL+"/api/sessions/"+created.ID, ""))
	if len(got.Turns) != 2 {
		t.Errorf("turns = %+v", got.Turns)
	}
}

func TestListAndDelete(t *testing.T) {
	srv, _ := newTestServer(t)
	a := decode[agentloop.SessionView](t, do(t, http.MethodPost, srv.URL+"/api/sessions", ""))
	time.Sleep(5 * time.Millisecond)
	b := decode[agentloop.SessionView](t, do(t, http.MethodPost, srv.URL+"/api/sessions", "{}"))

	list := decode[[]agentloop.SessionView](t, do(t, http.MethodGet, srv.URL+"/api/sessions", ""))
	if len(list) != 2 || list[0].ID != b.ID || list[1].ID != a.ID {
		t.Fatalf("list should be newest first: %+v", list)
	}

	if resp := do(t, http.MethodDelete, srv.URL+"/api/sessions/"+a.ID, ""); resp.StatusCode != http.StatusNoContent {
		t.Errorf("delete status %d", resp.StatusCode)
	}
	for _, path := range []string{"/api/sessions/" + a.ID, "/api/sessions/" + a.ID + "/events"} {
		if resp := do(t, http.MethodGet, srv.URL+path, ""); resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s after delete status %d", path, resp.StatusCode)
		}
	}
	if list := decode[[]agentloop.SessionView](t, do(t, http.MethodGet, srv.URL+"/api/sessions", "")); len(list) != 1 {
		t.Errorf("list after delete = %+v", list)
	}
}

func TestBadRequests(t *testing.T) {
	srv, _ := newTestServer(t)
	s := decode[agentloop.SessionView](t, do(t, http.MethodPost, srv.URL+"/api/sessions", ""))
	cases := []struct {
		method, path, body string
		want               int
	}{
		{http.MethodPost, "/api/sessions", "{bad", http.StatusBadRequest},
		{http.MethodPost, "/api/sessions/" + s.ID + "/messages", `{"prompt":"  "}`, http.StatusBadRequest},
		{http.MethodPost, "/api/sessions/nope/messages", `{"prompt":"hi"}`, http.StatusNotFound},
		{http.MethodPost, "/api/sessions/" + s.ID + "/permissions/p9", `{"allow":true}`, http.StatusNotFound},
		{http.MethodPost, "/api/sessions/" + s.ID + "/permissions/p9", `nope`, http.StatusBadRequest},
		{http.MethodDelete, "/api/sessions/nope", "", http.StatusNotFound},
	}
	for _, c := range cases {
		resp := do(t, c.method, srv.URL+c.path, c.body)
		if resp.StatusCode != c.want {
			t.Errorf("%s %s: status %d, want %d", c.method, c.path, resp.StatusCode, c.want)
		}
		if resp.StatusCode >= 400 {
			if body := decode[map[string]string](t, resp); body["error"] == "" {
				t.Errorf("%s %s: no error message", c.method, c.path)
			}
		}
	}
}

func TestStatusOf(t *testing.T) {
	if statusOf(agentloop.ErrSessionBusy) != http.StatusConflict || statusOf(agentloop.ErrSessionNotFound) != http.StatusNotFound ||
		statusOf(errors.New("x")) != http.StatusBadRequest {
		t.Error("unexpected status mapping")
	}
}
