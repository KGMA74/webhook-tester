package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// newTestServer builds a minimal Server suitable for unit testing.
// It wires a trivial template so tests are not coupled to the real HTML.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	tfs := fstest.MapFS{
		"template/index.html": &fstest.MapFile{
			Data: []byte(`count:{{.Count}}`),
		},
	}
	tmpl, err := template.New("index.html").
		Funcs(template.FuncMap{"join": strings.Join}).
		ParseFS(tfs, "template/index.html")
	if err != nil {
		t.Fatalf("parse test template: %v", err)
	}
	return &Server{store: NewWebhookStore(), tmpl: tmpl}
}

func TestHandleIndex_EmptyStore(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	s.handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if got := w.Body.String(); got != "count:0" {
		t.Errorf("unexpected body: %q", got)
	}
}

func TestHandleWebhook_CapturesJSONBody(t *testing.T) {
	s := newTestServer(t)
	payload := `{"event":"payment.success","amount":5000}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	reqs := s.store.GetAll()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 stored request, got %d", len(reqs))
	}
	// Body must be pretty-printed JSON
	if !strings.Contains(reqs[0].Body, "\n") {
		t.Errorf("expected pretty-printed JSON body, got: %q", reqs[0].Body)
	}
	if reqs[0].Method != http.MethodPost {
		t.Errorf("expected method POST, got %q", reqs[0].Method)
	}
}

func TestHandleWebhook_RespectsBoundary(t *testing.T) {
	s := newTestServer(t)
	w := httptest.NewRecorder()

	for i := 0; i < maxRequests+5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{}`))
		s.handleWebhook(w, req)
	}

	if got := len(s.store.GetAll()); got != maxRequests {
		t.Errorf("store should be capped at %d, got %d", maxRequests, got)
	}
}

func TestHandleClear_EmptiesStore(t *testing.T) {
	s := newTestServer(t)

	// Seed one entry
	seed := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"x":1}`))
	s.handleWebhook(httptest.NewRecorder(), seed)

	if len(s.store.GetAll()) != 1 {
		t.Fatal("precondition: store should have 1 entry")
	}

	req := httptest.NewRequest(http.MethodPost, "/clear", nil)
	w := httptest.NewRecorder()
	s.handleClear(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected 303 redirect, got %d", w.Code)
	}
	if len(s.store.GetAll()) != 0 {
		t.Errorf("store should be empty after clear")
	}
}

func TestHandleAPIRequests_ReturnsJSON(t *testing.T) {
	s := newTestServer(t)

	// Seed two entries
	for _, p := range []string{`{"a":1}`, `{"b":2}`} {
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(p))
		s.handleWebhook(httptest.NewRecorder(), req)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/requests", nil)
	w := httptest.NewRecorder()
	s.handleAPIRequests(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var out []WebhookRequest
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("expected 2 items in JSON response, got %d", len(out))
	}
}
