package upstream

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Model struct {
	ID string `json:"id"`
}

func FetchModels(baseURL, apiKey string) ([]string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("Base URL 为空")
	}
	raw, status, err := get(baseURL+"/models", apiKey, 20*time.Second)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("上游 %d: %s", status, truncate(string(raw), 240))
	}
	var wrap struct {
		Data []Model `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, fmt.Errorf("解析 /v1/models 失败: %w", err)
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(wrap.Data))
	for _, m := range wrap.Data {
		id := strings.TrimSpace(m.ID)
		if id == "" || strings.Contains(id, "/") {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func Probe(baseURL, apiKey string) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return fmt.Errorf("Base URL 为空")
	}
	health := strings.TrimSuffix(baseURL, "/v1") + "/health"
	if _, status, err := get(health, "", 3*time.Second); err == nil && status < 500 {
		return nil
	}
	_, status, err := get(baseURL+"/models", apiKey, 4*time.Second)
	if err != nil {
		return err
	}
	if status >= 500 {
		return fmt.Errorf("上游 %d", status)
	}
	return nil
}

func get(endpoint, apiKey string, timeout time.Duration) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	}
	if u, err := url.Parse(endpoint); err == nil && u.Host != "" {
		req.Host = u.Host
		req.Header.Set("Host", u.Host)
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	return raw, resp.StatusCode, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
