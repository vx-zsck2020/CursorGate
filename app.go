package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"CursorGate/internal/cursor"
	"CursorGate/internal/mux"
	"CursorGate/internal/native"
	"CursorGate/internal/store"
	"CursorGate/internal/update"
	"CursorGate/internal/upstream"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var appRef *App

type App struct {
	ctx       context.Context
	store     *store.Store
	mux       *mux.Server
	mu        sync.Mutex
	forceQuit bool
	appVer    update.Status
}

type ProviderView struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	BaseURL   string   `json:"baseUrl"`
	Enabled   bool     `json:"enabled"`
	CatchAll  bool     `json:"catchAll"`
	HasKey    bool     `json:"hasKey"`
	Online    bool     `json:"online"`
	OnlineErr string   `json:"onlineErr"`
	Models    []string `json:"models"`
	LastError string   `json:"lastError"`
}

type Dashboard struct {
	Providers []ProviderView `json:"providers"`
	Gateway   mux.Status     `json:"gateway"`
	Cursor    cursor.Status  `json:"cursor"`
	Prefs     store.Prefs    `json:"prefs"`
	App       update.Status  `json:"app"`
}

type SaveRequest struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	BaseURL  string `json:"baseUrl"`
	APIKey   string `json:"apiKey"`
	Enabled  bool   `json:"enabled"`
	CatchAll bool   `json:"catchAll"`
}

type CloseDecision struct {
	Action   string `json:"action"`
	Remember bool   `json:"remember"`
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	appRef = a
	path, err := store.DefaultPath()
	if err != nil {
		return
	}
	st, err := store.Open(path)
	if err != nil {
		return
	}
	a.store = st
	a.mux = mux.New(st)
	_, _ = a.mux.Start()
	a.applyNativeTheme(st.Prefs().Theme)
	go a.autoCheckUpdate()
}

func (a *App) domReady(ctx context.Context) {
	native.LockClientSize(appTitle, windowWidth, windowHeight)
	w, h := runtime.WindowGetSize(ctx)
	if w > 0 && h > 0 {
		runtime.WindowSetMinSize(ctx, w, h)
		runtime.WindowSetMaxSize(ctx, w, h)
	}
	native.StartTray(appTitle, native.TrayHooks{
		Show: a.ShowWindow,
		Quit: func() { _ = a.QuitApp() },
	})
}

func (a *App) shutdown(ctx context.Context) {
	native.DestroyTray()
	if a.mux != nil {
		_ = a.mux.Stop()
	}
}

func (a *App) beforeClose(ctx context.Context) bool {
	a.mu.Lock()
	force := a.forceQuit
	a.mu.Unlock()
	if force {
		return false
	}
	if a.store != nil {
		p := a.store.Prefs()
		if p.RememberClose {
			switch p.CloseAction {
			case "quit":
				return false
			case "tray":
				runtime.WindowHide(ctx)
				return true
			}
		}
	}
	runtime.EventsEmit(ctx, "cursorgate:ask-close")
	return true
}

func (a *App) GetDashboard() (Dashboard, error) {
	if err := a.ready(); err != nil {
		return Dashboard{}, err
	}
	list := a.store.List()
	health := probeAll(a.store, list)
	views := make([]ProviderView, 0, len(list))
	for _, p := range list {
		v := toView(p)
		if h, ok := health[p.ID]; ok {
			v.Online = h.ok
			v.OnlineErr = h.err
		}
		views = append(views, v)
	}
	cs, _ := cursor.ReadStatus()
	a.mu.Lock()
	appSt := a.appVer
	a.mu.Unlock()
	if appSt.Current == "" {
		appSt.Current = update.AppVersion
	}
	return Dashboard{
		Providers: views,
		Gateway:   a.mux.Status(),
		Cursor:    cs,
		Prefs:     a.store.Prefs(),
		App:       appSt,
	}, nil
}

func (a *App) SaveProvider(req SaveRequest) (ProviderView, error) {
	if err := a.ready(); err != nil {
		return ProviderView{}, err
	}
	p := store.Provider{
		ID:       req.ID,
		Name:     req.Name,
		BaseURL:  req.BaseURL,
		Enabled:  req.Enabled,
		CatchAll: req.CatchAll,
	}
	saved, err := a.store.Upsert(p, req.APIKey)
	if err != nil {
		return ProviderView{}, err
	}
	return toView(saved), nil
}

func (a *App) DeleteProvider(id string) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.store.Delete(id)
}

func (a *App) SetProviderEnabled(id string, enabled bool) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.store.SetEnabled(id, enabled)
}

func (a *App) RefreshModels(id string) ([]string, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	p, ok := a.store.Get(id)
	if !ok {
		return nil, fmt.Errorf("找不到该 API")
	}
	key, err := a.store.DecryptKey(p)
	if err != nil {
		_ = a.store.SetModels(id, nil, err.Error())
		return nil, err
	}
	models, err := upstream.FetchModels(p.BaseURL, key)
	if err != nil {
		_ = a.store.SetModels(id, p.Models, err.Error())
		return nil, err
	}
	if err := a.store.SetModels(id, models, ""); err != nil {
		return nil, err
	}
	return models, nil
}

func (a *App) StartGateway() (mux.Status, error) {
	if err := a.ready(); err != nil {
		return mux.Status{}, err
	}
	return a.mux.Start()
}

func (a *App) StopGateway() error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.mux.Stop()
}

func (a *App) ApplyToCursor() (cursor.Status, error) {
	if err := a.ready(); err != nil {
		return cursor.Status{}, err
	}
	st := a.mux.Status()
	if !st.Running {
		if _, err := a.mux.Start(); err != nil {
			return cursor.Status{}, err
		}
		st = a.mux.Status()
	}
	merged, err := mux.MergeModels(a.store)
	if err != nil {
		return cursor.Status{}, err
	}
	if len(merged) == 0 {
		return cursor.Status{}, fmt.Errorf("没有可同步的模型，请先拉取各 API 的模型列表")
	}
	wb := cursor.InspectWorkbench()
	if !wb.Patchable {
		return cursor.Status{}, fmt.Errorf("当前 Cursor %s 无法适配：%s", cursor.Version(), wb.Reason)
	}
	wasRunning := cursor.Running()
	if wasRunning {
		if err := cursor.Quit(20 * time.Second); err != nil {
			return cursor.Status{}, fmt.Errorf("无法退出 Cursor：%w", err)
		}
	}
	base := "http://" + st.Addr + "/v1"
	if err := cursor.EnsureCatalogRefreshSkip(); err != nil {
		if wasRunning {
			_ = cursor.Launch()
		}
		return cursor.Status{}, fmt.Errorf("无法短路官方模型刷新：%w", err)
	}
	if err := cursor.ApplyModels(base, merged); err != nil {
		if wasRunning {
			_ = cursor.Launch()
		}
		return cursor.Status{}, err
	}
	if err := cursor.Launch(); err != nil {
		cs, _ := cursor.ReadStatus()
		return cs, fmt.Errorf("已写入，但启动 Cursor 失败：%w", err)
	}
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		if cursor.Running() {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	return cursor.ReadStatus()
}

func (a *App) RestoreCursor() (cursor.RestoreResult, error) {
	if err := a.ready(); err != nil {
		return cursor.RestoreResult{}, err
	}
	return cursor.RestoreEnvironment()
}

func (a *App) SetTheme(theme string) (store.Prefs, error) {
	if err := a.ready(); err != nil {
		return store.Prefs{}, err
	}
	if err := a.store.SetTheme(theme); err != nil {
		return store.Prefs{}, err
	}
	p := a.store.Prefs()
	a.applyNativeTheme(p.Theme)
	return p, nil
}

func (a *App) ConfirmClose(dec CloseDecision) error {
	if a.store != nil && dec.Remember {
		_ = a.store.SetCloseBehavior(dec.Action, true)
	}
	if dec.Action == "tray" {
		if a.ctx != nil {
			runtime.WindowHide(a.ctx)
		}
		return nil
	}
	return a.QuitApp()
}

func (a *App) OpenCursorDownload() {
	if a.ctx != nil {
		runtime.BrowserOpenURL(a.ctx, cursor.DownloadURL)
	}
}

func (a *App) CheckUpdate() (update.Status, error) {
	st, err := update.Check()
	if st.Current == "" {
		st.Current = update.AppVersion
	}
	a.mu.Lock()
	a.appVer = st
	a.mu.Unlock()
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "cursorgate:update", st)
	}
	return st, err
}

func (a *App) ApplyUpdate() error {
	st, err := update.Check()
	if err != nil {
		return err
	}
	if !st.Available {
		return fmt.Errorf("当前已是最新版 %s", st.Current)
	}
	if err := update.Apply(st); err != nil {
		return err
	}
	return a.QuitApp()
}

func (a *App) OpenReleasePage() {
	st, _ := update.Check()
	url := st.URL
	if url == "" {
		url = "https://github.com/" + update.RepoOwner + "/" + update.RepoName + "/releases/latest"
	}
	if a.ctx != nil {
		runtime.BrowserOpenURL(a.ctx, url)
	}
}

func (a *App) autoCheckUpdate() {
	time.Sleep(1200 * time.Millisecond)
	_, _ = a.CheckUpdate()
}

func (a *App) ShowWindow() {
	if a.ctx == nil {
		return
	}
	runtime.WindowShow(a.ctx)
	runtime.WindowUnminimise(a.ctx)
}

func (a *App) QuitApp() error {
	a.mu.Lock()
	a.forceQuit = true
	a.mu.Unlock()
	native.DestroyTray()
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
	return nil
}

func (a *App) applyNativeTheme(theme string) {
	if a.ctx == nil {
		return
	}
	if theme == "light" {
		runtime.WindowSetLightTheme(a.ctx)
		return
	}
	runtime.WindowSetDarkTheme(a.ctx)
}

func (a *App) ready() error {
	if a.store == nil || a.mux == nil {
		return fmt.Errorf("应用尚未就绪")
	}
	return nil
}

func toView(p store.Provider) ProviderView {
	return ProviderView{
		ID:        p.ID,
		Name:      p.Name,
		BaseURL:   p.BaseURL,
		Enabled:   p.Enabled,
		CatchAll:  p.CatchAll,
		HasKey:    strings.TrimSpace(p.KeyEnc) != "",
		Models:    p.Models,
		LastError: p.LastError,
	}
}

type probeResult struct {
	ok  bool
	err string
}

func probeAll(st *store.Store, list []store.Provider) map[string]probeResult {
	out := make(map[string]probeResult, len(list))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, p := range list {
		p := p
		wg.Add(1)
		go func() {
			defer wg.Done()
			res := probeResult{}
			key, err := st.DecryptKey(p)
			if err != nil {
				res.err = err.Error()
			} else if err := upstream.Probe(p.BaseURL, key); err != nil {
				res.err = err.Error()
			} else {
				res.ok = true
			}
			mu.Lock()
			out[p.ID] = res
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out
}
