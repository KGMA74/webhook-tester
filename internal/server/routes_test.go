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

func newTestServer(t *testing.T) (*Server, *Endpoint) {
	t.Helper()
	tfs := fstest.MapFS{
		"template/index.html": &fstest.MapFile{Data: []byte(`ok`)},
		"template/home.html":  &fstest.MapFile{Data: []byte(`ok`)},
	}
	tmpls, err := template.New("").ParseFS(tfs, "template/*.html")
	if err != nil {
		t.Fatalf("parse test template: %v", err)
	}
	s := &Server{registry: newEndpointRegistry(), tmpls: tmpls}
	ep := s.registry.Create()
	return s, ep
}

func TestHandleHealth(t *testing.T) {
	s, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	s.handleHealth(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandleIndex(t *testing.T) {
	s, ep := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/"+ep.ID, nil)
	req.SetPathValue("id", ep.ID)
	w := httptest.NewRecorder()
	s.handleIndex(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("expected text/html content-type, got %q", ct)
	}
}

func TestHandleIndex_UnknownID(t *testing.T) {
	s, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/doesnotexist", nil)
	req.SetPathValue("id", "doesnotexist")
	w := httptest.NewRecorder()
	s.handleIndex(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleWebhook_CapturesJSONBody(t *testing.T) {
	s, ep := newTestServer(t)
	payload := `{"event":"payment.success","amount":5000}`
	req := httptest.NewRequest(http.MethodPost, "/hook/"+ep.ID, strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", ep.ID)
	w := httptest.NewRecorder()

	s.handleWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	reqs := ep.store.GetAll()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 stored request, got %d", len(reqs))
	}
	if !strings.Contains(reqs[0].Body, "\n") {
		t.Errorf("body should be pretty-printed JSON, got: %q", reqs[0].Body)
	}
}

func TestHandleWebhook_CapturesQueryParams(t *testing.T) {
	s, ep := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/hook/"+ep.ID+"?event=payment.success&token=abc", strings.NewReader(`{}`))
	req.SetPathValue("id", ep.ID)
	s.handleWebhook(httptest.NewRecorder(), req)

	reqs := ep.store.GetAll()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if reqs[0].Query == nil || reqs[0].Query["event"][0] != "payment.success" {
		t.Errorf("query params not captured: %v", reqs[0].Query)
	}
}

func TestHandleWebhook_RespectsBoundary(t *testing.T) {
	s, ep := newTestServer(t)
	for i := 0; i < maxRequests+10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/hook/"+ep.ID, strings.NewReader(`{}`))
		req.SetPathValue("id", ep.ID)
		s.handleWebhook(httptest.NewRecorder(), req)
	}
	if got := len(ep.store.GetAll()); got != maxRequests {
		t.Errorf("store should be capped at %d, got %d", maxRequests, got)
	}
}

func TestHandleClear_EmptiesStore(t *testing.T) {
	s, ep := newTestServer(t)
	seed := httptest.NewRequest(http.MethodPost, "/hook/"+ep.ID, strings.NewReader(`{"x":1}`))
	seed.SetPathValue("id", ep.ID)
	s.handleWebhook(httptest.NewRecorder(), seed)
	if len(ep.store.GetAll()) != 1 {
		t.Fatal("precondition: store should have 1 entry")
	}

	req := httptest.NewRequest(http.MethodPost, "/api/"+ep.ID+"/clear", nil)
	req.SetPathValue("id", ep.ID)
	s.handleClear(httptest.NewRecorder(), req)

	if len(ep.store.GetAll()) != 0 {
		t.Error("store should be empty after clear")
	}
}

func TestHandleAPIRequests_ReturnsJSON(t *testing.T) {
	s, ep := newTestServer(t)
	for _, p := range []string{`{"a":1}`, `{"b":2}`} {
		req := httptest.NewRequest(http.MethodPost, "/hook/"+ep.ID, strings.NewReader(p))
		req.SetPathValue("id", ep.ID)
		s.handleWebhook(httptest.NewRecorder(), req)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/"+ep.ID+"/requests", nil)
	req.SetPathValue("id", ep.ID)
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
		t.Errorf("expected 2 items, got %d", len(out))
	}
}

func TestRequireAuth_BlocksWithoutToken(t *testing.T) {
	s, _ := newTestServer(t)
	s.token = "secret"

	dummy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	s.requireAuth(dummy)(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRequireAuth_AllowsCorrectToken(t *testing.T) {
	s, _ := newTestServer(t)
	s.token = "secret"

	dummy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("any", "secret")
	w := httptest.NewRecorder()
	s.requireAuth(dummy)(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
