package mux

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"CursorGate/internal/store"
)

func TestPeekModel(t *testing.T) {
	if got := peekModel([]byte(`{"model":"hy3","stream":true}`)); got != "hy3" {
		t.Fatalf("got %q", got)
	}
}

func TestMergeAndRoute(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	wb, err := st.Upsert(store.Provider{
		Name:    "WorkBuddy",
		BaseURL: "http://127.0.0.1:8899/v1",
		Enabled: true,
		Models:  []string{"deepseek-v4.1-flash", "hy3", "hy4-preview-f", "x-ai/skip-me"},
	}, "sk-wb-test")
	if err != nil {
		t.Fatal(err)
	}
	dct, err := st.Upsert(store.Provider{
		Name:     "DCT",
		BaseURL:  "https://dct.example/v1",
		Enabled:  true,
		CatchAll: true,
		Models:   []string{"composer-2.5", "hy3"},
	}, "sk-dct-test")
	if err != nil {
		t.Fatal(err)
	}
	_ = dct
	merged, err := MergeModels(st)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 4 {
		t.Fatalf("merged=%v", merged)
	}
	for _, id := range merged {
		if id == "x-ai/skip-me" {
			t.Fatal("slash alias should be dropped")
		}
	}
	p, key, err := Route(st, "hy3")
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != wb.ID || key != "sk-wb-test" {
		t.Fatalf("hy3 should hit WorkBuddy first, got %s %s", p.Name, key)
	}
	p, key, err = Route(st, "composer-2.5")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "DCT" || key != "sk-dct-test" {
		t.Fatalf("composer should hit DCT, got %s", p.Name)
	}
	p, _, err = Route(st, "unknown-new")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "WorkBuddy" {
		t.Fatalf("unknown should fall to first enabled API, got %s", p.Name)
	}
}

func TestListenFallsBackWhenPreferredBusy(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	port := busy.Addr().(*net.TCPAddr).Port
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetPort(port); err != nil {
		t.Fatal(err)
	}
	s := New(st)
	stt, err := s.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	if !stt.Running {
		t.Fatal("gateway should start")
	}
	if stt.Port == 0 {
		t.Fatal("status should report listen port")
	}
	gotPort := strings.Split(stt.Addr, ":")
	if gotPort[len(gotPort)-1] == fmt.Sprintf("%d", port) {
		t.Fatalf("still bound to busy port %s", stt.Addr)
	}
	if stt.Port == port {
		t.Fatalf("status port still busy %d", stt.Port)
	}
	if st.Port() == port {
		t.Fatal("preferred busy port should not be persisted")
	}
}

func TestHealth(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	s := New(st)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	s.routes().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestModelsAuth(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.Upsert(store.Provider{
		Name:    "WB",
		BaseURL: "http://127.0.0.1:8899/v1",
		Enabled: true,
		Models:  []string{"hy3"},
	}, "sk-ok")
	if err != nil {
		t.Fatal(err)
	}
	s := New(st)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	s.routes().ServeHTTP(rr, req)
	if rr.Code != 401 {
		t.Fatalf("want 401 got %d", rr.Code)
	}
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer sk-ok")
	s.routes().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("want 200 got %d %s", rr.Code, rr.Body.String())
	}
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data) != 1 || body.Data[0].ID != "hy3" {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestProxyForwardsModel(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if r.Header.Get("Authorization") != "Bearer sk-up" {
			http.Error(w, "bad auth "+r.Header.Get("Authorization"), 401)
			return
		}
		if !bytes.Contains(raw, []byte(`"hy3"`)) {
			http.Error(w, "missing model", 400)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"ok\":true}\n\n"))
	}))
	defer upstream.Close()

	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.Upsert(store.Provider{
		Name:    "WB",
		BaseURL: upstream.URL + "/v1",
		Enabled: true,
		Models:  []string{"hy3"},
	}, "sk-up")
	if err != nil {
		t.Fatal(err)
	}
	s := New(st)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader([]byte(`{"model":"hy3"}`)))
	req.Header.Set("Authorization", "Bearer sk-up")
	s.routes().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte(`{"ok":true}`)) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}
