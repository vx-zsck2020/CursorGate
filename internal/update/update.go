package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	AppVersion = "1.0.0"
	RepoOwner  = "vx-zsck2020"
	RepoName   = "CursorGate"
	UserAgent  = "CursorGate/" + AppVersion
)

type Status struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Notes     string `json:"notes"`
	URL       string `json:"url"`
	AssetName string `json:"assetName"`
	AssetURL  string `json:"assetUrl"`
	SHA256    string `json:"sha256,omitempty"`
	Available bool   `json:"available"`
	Error     string `json:"error,omitempty"`
}

type ghRelease struct {
	TagName    string    `json:"tag_name"`
	Name       string    `json:"name"`
	Body       string    `json:"body"`
	HTMLURL    string    `json:"html_url"`
	Prerelease bool      `json:"prerelease"`
	Draft      bool      `json:"draft"`
	Assets     []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type manifest struct {
	Version     string `json:"version"`
	Notes       string `json:"notes"`
	URL         string `json:"url"`
	DownloadURL string `json:"downloadUrl"`
	Asset       string `json:"asset"`
	SHA256      string `json:"sha256"`
}

func Check() (Status, error) {
	st := Status{Current: AppVersion}
	rel, err := fetchLatestRelease()
	if err == nil && rel != nil && !rel.Draft {
		st.Latest = normalizeVersion(rel.TagName)
		st.Notes = strings.TrimSpace(rel.Body)
		st.URL = rel.HTMLURL
		if a := pickAsset(rel.Assets); a != nil {
			st.AssetName = a.Name
			st.AssetURL = a.BrowserDownloadURL
			st.SHA256 = findSHA256(rel.Assets, a.Name)
		}
		st.Available = Newer(st.Latest, st.Current) && st.AssetURL != ""
		return st, nil
	}
	man, manErr := fetchManifest()
	if manErr != nil {
		if err != nil {
			st.Error = err.Error()
			return st, err
		}
		st.Error = manErr.Error()
		return st, manErr
	}
	st.Latest = normalizeVersion(man.Version)
	st.Notes = strings.TrimSpace(man.Notes)
	st.URL = man.URL
	if st.URL == "" {
		st.URL = releasePage()
	}
	st.AssetName = man.Asset
	st.AssetURL = man.DownloadURL
	st.SHA256 = strings.TrimSpace(man.SHA256)
	st.Available = Newer(st.Latest, st.Current) && st.AssetURL != ""
	return st, nil
}

func fetchLatestRelease() (*ghRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", RepoOwner, RepoName)
	raw, code, err := get(url)
	if err != nil {
		return nil, err
	}
	if code == 404 {
		return nil, fmt.Errorf("暂无发布")
	}
	if code != 200 {
		return nil, fmt.Errorf("GitHub API %d", code)
	}
	var rel ghRelease
	if err := json.Unmarshal(raw, &rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

func fetchManifest() (manifest, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/internal/update/latest.json", RepoOwner, RepoName)
	raw, code, err := get(url)
	if err != nil {
		return manifest{}, err
	}
	if code != 200 {
		return manifest{}, fmt.Errorf("latest.json %d", code)
	}
	var man manifest
	if err := json.Unmarshal(raw, &man); err != nil {
		return manifest{}, err
	}
	if strings.TrimSpace(man.Version) == "" {
		return manifest{}, fmt.Errorf("latest.json 缺少 version")
	}
	return man, nil
}

var getFn = defaultGet

func get(url string) ([]byte, int, error) {
	return getFn(url)
}

func defaultGet(url string) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return raw, resp.StatusCode, err
}

func pickAsset(assets []ghAsset) *ghAsset {
	var fallback *ghAsset
	for i := range assets {
		name := strings.ToLower(assets[i].Name)
		if !strings.HasSuffix(name, ".exe") {
			continue
		}
		a := &assets[i]
		if strings.Contains(name, "cursorgate") {
			return a
		}
		if fallback == nil {
			fallback = a
		}
	}
	return fallback
}

func findSHA256(assets []ghAsset, exeName string) string {
	want := strings.ToLower(exeName + ".sha256")
	for _, a := range assets {
		if strings.ToLower(a.Name) == want {
			raw, code, err := get(a.BrowserDownloadURL)
			if err != nil || code != 200 {
				return ""
			}
			line := strings.TrimSpace(string(raw))
			if i := strings.IndexByte(line, ' '); i > 0 {
				line = line[:i]
			}
			if i := strings.IndexByte(line, '\t'); i > 0 {
				line = line[:i]
			}
			return strings.ToLower(strings.TrimSpace(line))
		}
	}
	return ""
}

func DownloadTo(url, dest, wantSHA string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", UserAgent)
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("下载失败 HTTP %d", resp.StatusCode)
	}
	sum := sha256.New()
	tmp := dest + ".part"
	f, err := createFile(tmp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(io.MultiWriter(f, sum), io.LimitReader(resp.Body, 200<<20))
	closeErr := f.Close()
	if copyErr != nil {
		_ = removeFile(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = removeFile(tmp)
		return closeErr
	}
	got := hex.EncodeToString(sum.Sum(nil))
	if wantSHA != "" && !strings.EqualFold(wantSHA, got) {
		_ = removeFile(tmp)
		return fmt.Errorf("校验失败")
	}
	if err := replaceFile(tmp, dest); err != nil {
		_ = removeFile(tmp)
		return err
	}
	return nil
}

func Newer(latest, current string) bool {
	a := parseVersion(latest)
	b := parseVersion(current)
	n := len(a)
	if len(b) > n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		var av, bv int
		if i < len(a) {
			av = a[i]
		}
		if i < len(b) {
			bv = b[i]
		}
		if av != bv {
			return av > bv
		}
	}
	return false
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	if i := strings.IndexAny(v, " -+"); i >= 0 {
		v = v[:i]
	}
	return v
}

func parseVersion(v string) []int {
	v = normalizeVersion(v)
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, _ := strconv.Atoi(p)
		out = append(out, n)
	}
	return out
}

func releasePage() string {
	return fmt.Sprintf("https://github.com/%s/%s/releases/latest", RepoOwner, RepoName)
}
