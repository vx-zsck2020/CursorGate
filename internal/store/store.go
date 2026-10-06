package store

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"CursorGate/internal/secret"
)

type Provider struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	BaseURL   string    `json:"baseUrl"`
	KeyEnc    string    `json:"keyEnc"`
	Enabled   bool      `json:"enabled"`
	CatchAll  bool      `json:"catchAll"`
	Models    []string  `json:"models"`
	LastError string    `json:"lastError,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Prefs struct {
	Theme         string `json:"theme"`
	CloseAction   string `json:"closeAction"`
	RememberClose bool   `json:"rememberClose"`
}

type File struct {
	GatewayPort   int        `json:"gatewayPort"`
	Theme         string     `json:"theme"`
	CloseAction   string     `json:"closeAction"`
	RememberClose bool       `json:"rememberClose"`
	Providers     []Provider `json:"providers"`
}

type Store struct {
	mu   sync.Mutex
	path string
	data File
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "CursorGate", "config.json"), nil
}

func Open(path string) (*Store, error) {
	s := &Store{path: path, data: File{GatewayPort: 8900}}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				return nil, err
			}
			normalizePrefs(&s.data)
			return s, s.Save()
		}
		return nil, err
	}
	if err := json.Unmarshal(raw, &s.data); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if s.data.GatewayPort <= 0 {
		s.data.GatewayPort = 8900
	}
	normalizePrefs(&s.data)
	return s, nil
}

func normalizePrefs(f *File) {
	if f.Theme != "light" {
		f.Theme = "dark"
	}
	if f.CloseAction != "quit" && f.CloseAction != "tray" {
		f.CloseAction = ""
	}
}

func (s *Store) Save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) Snapshot() File {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := s.data
	cp.Providers = append([]Provider(nil), s.data.Providers...)
	return cp
}

func (s *Store) Port() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.GatewayPort
}

func (s *Store) SetPort(port int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if port <= 0 || port > 65535 {
		return fmt.Errorf("非法端口")
	}
	if s.data.GatewayPort == port {
		return nil
	}
	s.data.GatewayPort = port
	return s.Save()
}

func (s *Store) Prefs() Prefs {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Prefs{
		Theme:         s.data.Theme,
		CloseAction:   s.data.CloseAction,
		RememberClose: s.data.RememberClose,
	}
}

func (s *Store) SetTheme(theme string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if theme != "light" {
		theme = "dark"
	}
	s.data.Theme = theme
	return s.Save()
}

func (s *Store) SetCloseBehavior(action string, remember bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if action != "quit" && action != "tray" {
		action = ""
	}
	s.data.CloseAction = action
	s.data.RememberClose = remember && action != ""
	return s.Save()
}

func (s *Store) List() []Provider {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Provider, len(s.data.Providers))
	copy(out, s.data.Providers)
	return out
}

func (s *Store) Get(id string) (Provider, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.data.Providers {
		if p.ID == id {
			return p, true
		}
	}
	return Provider{}, false
}

func (s *Store) Upsert(p Provider, newKey string) (Provider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.Name = strings.TrimSpace(p.Name)
	p.BaseURL = strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if p.Name == "" {
		return Provider{}, fmt.Errorf("名称不能为空")
	}
	if p.BaseURL == "" {
		return Provider{}, fmt.Errorf("Base URL 不能为空")
	}
	if !strings.HasSuffix(p.BaseURL, "/v1") {
		p.BaseURL += "/v1"
	}
	if p.ID == "" {
		p.ID = newID()
		if strings.TrimSpace(newKey) == "" {
			return Provider{}, fmt.Errorf("新 API 必须填写密钥")
		}
	}
	if strings.TrimSpace(newKey) != "" {
		enc, err := secret.Protect([]byte(strings.TrimSpace(newKey)))
		if err != nil {
			return Provider{}, err
		}
		p.KeyEnc = base64.StdEncoding.EncodeToString(enc)
	}
	p.UpdatedAt = time.Now()
	found := false
	for i, old := range s.data.Providers {
		if old.ID == p.ID {
			if p.KeyEnc == "" {
				p.KeyEnc = old.KeyEnc
			}
			if p.Models == nil {
				p.Models = old.Models
			}
			s.data.Providers[i] = p
			found = true
			break
		}
	}
	if !found {
		s.data.Providers = append(s.data.Providers, p)
	}
	if err := s.Save(); err != nil {
		return Provider{}, err
	}
	return p, nil
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.data.Providers[:0]
	for _, p := range s.data.Providers {
		if p.ID != id {
			out = append(out, p)
		}
	}
	s.data.Providers = out
	return s.Save()
}

func (s *Store) SetEnabled(id string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, p := range s.data.Providers {
		if p.ID == id {
			s.data.Providers[i].Enabled = enabled
			s.data.Providers[i].UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("找不到该 API")
}

func (s *Store) SetModels(id string, models []string, lastErr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, p := range s.data.Providers {
		if p.ID == id {
			s.data.Providers[i].Models = models
			s.data.Providers[i].LastError = lastErr
			s.data.Providers[i].UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("找不到该 API")
}

func (s *Store) DecryptKey(p Provider) (string, error) {
	if p.KeyEnc == "" {
		return "", fmt.Errorf("未保存密钥")
	}
	raw, err := base64.StdEncoding.DecodeString(p.KeyEnc)
	if err != nil {
		return "", err
	}
	plain, err := secret.Unprotect(raw)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
