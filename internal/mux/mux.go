package mux

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"CursorGate/internal/store"
)

type Server struct {
	store   *store.Store
	mu      sync.Mutex
	http    *http.Server
	ln      net.Listener
	cancel  context.CancelFunc
	lastErr string
}

type Status struct {
	Running bool   `json:"running"`
	Addr    string `json:"addr"`
	Port    int    `json:"port"`
	Error   string `json:"error,omitempty"`
}

func New(st *store.Store) *Server {
	return &Server{store: st}
}

func (s *Server) Start() (Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.http != nil {
		return s.statusLocked(), nil
	}
	prefer := s.store.Port()
	ln, err := listenLoopback(prefer)
	if err != nil {
		s.lastErr = err.Error()
		return Status{Addr: fmt.Sprintf("127.0.0.1:%d", prefer), Port: prefer, Error: s.lastErr}, err
	}
	if tcp, ok := ln.Addr().(*net.TCPAddr); ok && tcp.Port != prefer {
		_ = s.store.SetPort(tcp.Port)
	}
	ctx, cancel := context.WithCancel(context.Background())
	h := &http.Server{
		Handler:           s.routes(),
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	s.ln, s.http, s.cancel = ln, h, cancel
	s.lastErr = ""
	go func() { _ = h.Serve(ln) }()
	return s.statusLocked(), nil
}

func listenLoopback(prefer int) (net.Listener, error) {
	tried := map[int]struct{}{}
	try := func(port int) (net.Listener, error) {
		if port <= 0 || port > 65535 {
			return nil, fmt.Errorf("非法端口")
		}
		if _, ok := tried[port]; ok {
			return nil, fmt.Errorf("already tried")
		}
		tried[port] = struct{}{}
		return net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	}
	if ln, err := try(prefer); err == nil {
		return ln, nil
	}
	for p := 8900; p <= 8999; p++ {
		if ln, err := try(p); err == nil {
			return ln, nil
		}
	}
	return net.Listen("tcp", "127.0.0.1:0")
}

func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.http == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := s.http.Shutdown(ctx)
	if s.cancel != nil {
		s.cancel()
	}
	s.http, s.ln, s.cancel = nil, nil, nil
	s.lastErr = ""
	return err
}

func (s *Server) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.statusLocked()
}

func (s *Server) statusLocked() Status {
	port := s.store.Port()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	if s.ln == nil {
		return Status{Running: false, Addr: addr, Port: port, Error: s.lastErr}
	}
	if tcp, ok := s.ln.Addr().(*net.TCPAddr); ok {
		port = tcp.Port
		addr = s.ln.Addr().String()
	}
	return Status{Running: true, Addr: addr, Port: port, Error: s.lastErr}
}

func (s *Server) RoutesForTest() http.Handler {
	return s.routes()
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"ok": true, "name": "cursorgate-mux"})
	})
	mux.HandleFunc("/v1/models", s.handleModels)
	mux.HandleFunc("/v1/", s.handleProxy)
	return mux
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		cors(w)
		w.WriteHeader(204)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	if _, err := s.authorize(r); err != nil {
		writeJSON(w, 401, map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}
	merged, err := MergeModels(s.store)
	if err != nil {
		writeJSON(w, 502, map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}
	data := make([]map[string]any, 0, len(merged))
	for _, id := range merged {
		data = append(data, map[string]any{"id": id, "object": "model", "owned_by": "cursorgate"})
	}
	writeJSON(w, 200, map[string]any{"object": "list", "data": data})
}

func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		cors(w)
		w.WriteHeader(204)
		return
	}
	if _, err := s.authorize(r); err != nil {
		writeJSON(w, 401, map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	model := peekModel(body)
	p, key, err := Route(s.store, model)
	if err != nil {
		writeJSON(w, 400, map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}
	base := strings.TrimRight(p.BaseURL, "/")
	base = strings.TrimSuffix(base, "/v1")
	target := base + r.URL.Path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	req, err := http.NewRequestWithContext(r.Context(), r.Method, target, bytes.NewReader(body))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	copyHeaders(req.Header, r.Header)
	req.Header.Set("Authorization", "Bearer "+key)
	if u := req.URL; u != nil && u.Host != "" {
		req.Host = u.Host
		req.Header.Set("Host", u.Host)
	}
	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		writeJSON(w, 502, map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}
	defer resp.Body.Close()
	for k, vs := range resp.Header {
		if skipHeader(k) {
			continue
		}
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	flusher, _ := w.(http.Flusher)
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, _ = w.Write(buf[:n])
			if flusher != nil {
				flusher.Flush()
			}
		}
		if readErr != nil {
			break
		}
	}
}

func (s *Server) authorize(r *http.Request) (string, error) {
	got := bearer(r.Header.Get("Authorization"))
	if got == "" {
		got = strings.TrimSpace(r.Header.Get("X-Api-Key"))
	}
	if got == "" {
		return "", fmt.Errorf("缺少密钥")
	}
	for _, p := range s.store.List() {
		key, err := s.store.DecryptKey(p)
		if err != nil {
			continue
		}
		if subtleEq(got, key) {
			return got, nil
		}
	}
	return "", fmt.Errorf("密钥无效")
}

func MergeModels(st *store.Store) ([]string, error) {
	seen := map[string]struct{}{}
	var out []string
	for _, p := range st.List() {
		if !p.Enabled {
			continue
		}
		for _, id := range p.Models {
			id = strings.TrimSpace(id)
			if id == "" || strings.Contains(id, "/") {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out, nil
}

func Route(st *store.Store, model string) (store.Provider, string, error) {
	list := st.List()
	model = strings.TrimSpace(model)
	if model == "" {
		return store.Provider{}, "", fmt.Errorf("请求缺少 model")
	}
	var first *store.Provider
	var catchAll *store.Provider
	for i := range list {
		p := list[i]
		if !p.Enabled {
			continue
		}
		if first == nil {
			cp := p
			first = &cp
		}
		if p.CatchAll && catchAll == nil {
			cp := p
			catchAll = &cp
		}
		for _, id := range p.Models {
			if id == model {
				key, err := st.DecryptKey(p)
				return p, key, err
			}
		}
	}
	fallback := catchAll
	if fallback == nil {
		fallback = first
	}
	if fallback != nil {
		key, err := st.DecryptKey(*fallback)
		return *fallback, key, err
	}
	return store.Provider{}, "", fmt.Errorf("未知模型 %s，且没有启用的 API", model)
}

func peekModel(body []byte) string {
	var obj struct {
		Model string `json:"model"`
	}
	_ = json.Unmarshal(body, &obj)
	return obj.Model
}

func bearer(h string) string {
	h = strings.TrimSpace(h)
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return h
}

func copyHeaders(dst, src http.Header) {
	for k, vs := range src {
		if skipHeader(k) || strings.EqualFold(k, "Authorization") || strings.EqualFold(k, "Host") {
			continue
		}
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

func skipHeader(k string) bool {
	switch strings.ToLower(k) {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
		"te", "trailers", "transfer-encoding", "upgrade", "content-length":
		return true
	}
	return false
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	cors(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Api-Key")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
}

func subtleEq(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
