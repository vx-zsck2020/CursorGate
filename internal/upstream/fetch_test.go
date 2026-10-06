package upstream

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchModelsDropsSlashAlias(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-5.6"},{"id":"x-ai/skip"},{"id":"gpt-5.6"}]}`))
	}))
	defer s.Close()
	ids, err := FetchModels(s.URL+"/v1", "sk")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "gpt-5.6" {
		t.Fatalf("%v", ids)
	}
}

func TestProbeHealth(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer s.Close()
	if err := Probe(s.URL+"/v1", "sk"); err != nil {
		t.Fatal(err)
	}
}
