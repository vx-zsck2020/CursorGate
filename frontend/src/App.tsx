import {useEffect, useMemo, useState} from 'react'
import {Banner, Button, Checkbox, Input, Modal, Select, Space, Switch, Toast, Typography} from '@douyinfe/semi-ui'
import {IconMoon, IconPlus, IconSun} from '@douyinfe/semi-icons'
import './App.css'
import {
    ApplyToCursor,
    ApplyUpdate,
    CheckUpdate,
    ConfirmClose,
    DeleteProvider,
    GetDashboard,
    OpenCursorDownload,
    OpenReleasePage,
    RefreshModels,
    RestoreCursor,
    SaveProvider,
    SetProviderEnabled,
    SetTheme,
} from '../wailsjs/go/main/App'
import {EventsOn} from '../wailsjs/runtime/runtime'
import {main} from '../wailsjs/go/models'

type ProviderView = main.ProviderView
type Dashboard = main.Dashboard

const emptyForm = {
    id: '',
    name: '',
    baseUrl: '',
    apiKey: '',
    enabled: true,
}

function applyTheme(theme: string) {
    document.body.setAttribute('theme-mode', theme === 'light' ? 'light' : 'dark')
}

function App() {
    const [dash, setDash] = useState<Dashboard | null>(null)
    const [selected, setSelected] = useState<string>('')
    const [form, setForm] = useState({...emptyForm})
    const [busy, setBusy] = useState('')
    const [theme, setThemeState] = useState('dark')
    const [askClose, setAskClose] = useState(false)
    const [rememberClose, setRememberClose] = useState(false)
    const [appVer, setAppVer] = useState({current: '1.0.0', latest: '', available: false, error: '', notes: '', url: ''})

    const providers = dash?.providers ?? []
    const current = useMemo(
        () => providers.find(p => p.id === selected),
        [providers, selected]
    )

    async function reload(keepId?: string) {
        const next = await GetDashboard()
        setDash(next)
        const nextTheme = next.prefs?.theme === 'light' ? 'light' : 'dark'
        setThemeState(nextTheme)
        applyTheme(nextTheme)
        if (next.app) {
            setAppVer({
                current: next.app.current || '1.0.0',
                latest: next.app.latest || '',
                available: !!next.app.available,
                error: next.app.error || '',
                notes: next.app.notes || '',
                url: next.app.url || '',
            })
        }
        const id = (keepId !== undefined ? keepId : selected) || next.providers?.[0]?.id || ''
        setSelected(id)
        const p = next.providers?.find(x => x.id === id)
        if (p) {
            setForm({
                id: p.id,
                name: p.name,
                baseUrl: p.baseUrl,
                apiKey: '',
                enabled: p.enabled,
            })
        } else if (!id) {
            setForm({...emptyForm})
        }
    }

    useEffect(() => {
        reload().catch(err => Toast.error(String(err)))
        const timer = window.setInterval(() => {
            GetDashboard().then(next => {
                setDash(next)
                const nextTheme = next.prefs?.theme === 'light' ? 'light' : 'dark'
                setThemeState(nextTheme)
                applyTheme(nextTheme)
                if (next.app) {
                    setAppVer({
                        current: next.app.current || '1.0.0',
                        latest: next.app.latest || '',
                        available: !!next.app.available,
                        error: next.app.error || '',
                        notes: next.app.notes || '',
                        url: next.app.url || '',
                    })
                }
            }).catch(() => undefined)
        }, 4000)
        const offClose = EventsOn('cursorgate:ask-close', () => setAskClose(true))
        const offUpdate = EventsOn('cursorgate:update', (st: any) => {
            if (!st) return
            setAppVer({
                current: st.current || '1.0.0',
                latest: st.latest || '',
                available: !!st.available,
                error: st.error || '',
                notes: st.notes || '',
                url: st.url || '',
            })
            if (st.available) {
                Toast.info(`发现新版本 ${st.latest}`)
            }
        })
        return () => {
            window.clearInterval(timer)
            offClose()
            offUpdate()
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    function startNew() {
        setSelected('')
        setForm({...emptyForm, enabled: true})
    }

    function pick(p: ProviderView) {
        setSelected(p.id)
        setForm({
            id: p.id,
            name: p.name,
            baseUrl: p.baseUrl,
            apiKey: '',
            enabled: p.enabled,
        })
    }

    async function save() {
        setBusy('save')
        try {
            const saved = await SaveProvider({
                id: form.id,
                name: form.name,
                baseUrl: form.baseUrl,
                apiKey: form.apiKey,
                enabled: form.enabled,
                catchAll: false,
            })
            Toast.success('已保存')
            await reload(saved.id)
        } catch (err) {
            Toast.error(String(err))
        } finally {
            setBusy('')
        }
    }

    async function refresh() {
        if (!form.id) {
            Toast.warning('请先保存这条 API')
            return
        }
        setBusy('refresh')
        try {
            const models = await RefreshModels(form.id)
            Toast.success(`已拉取 ${models.length} 个模型`)
            await reload(form.id)
        } catch (err) {
            Toast.error(String(err))
            await reload(form.id)
        } finally {
            setBusy('')
        }
    }

    async function remove() {
        if (!form.id) return
        setBusy('del')
        try {
            await DeleteProvider(form.id)
            Toast.success('已删除')
            setSelected('')
            await reload('')
        } catch (err) {
            Toast.error(String(err))
        } finally {
            setBusy('')
        }
    }

    async function toggle(p: ProviderView, enabled: boolean) {
        try {
            await SetProviderEnabled(p.id, enabled)
            await reload(selected)
        } catch (err) {
            Toast.error(String(err))
        }
    }

    async function apply() {
        setBusy('apply')
        try {
            await ApplyToCursor()
            Toast.success('已同步并重启 Cursor')
            await reload(selected)
        } catch (err) {
            Toast.error(String(err))
            await reload(selected)
        } finally {
            setBusy('')
        }
    }

    async function restore() {
        setBusy('restore')
        try {
            const result = await RestoreCursor()
            Toast.success(result.message || '已恢复 Cursor 环境')
            await reload(selected)
        } catch (err) {
            Toast.error(String(err))
            await reload(selected)
        } finally {
            setBusy('')
        }
    }

    async function switchTheme() {
        const next = theme === 'light' ? 'dark' : 'light'
        try {
            const prefs = await SetTheme(next)
            const applied = prefs.theme === 'light' ? 'light' : 'dark'
            setThemeState(applied)
            applyTheme(applied)
        } catch (err) {
            Toast.error(String(err))
        }
    }

    async function applyUpdate() {
        setBusy('update')
        try {
            await ApplyUpdate()
            Toast.success('已开始更新，程序即将重启')
        } catch (err) {
            Toast.error(String(err))
            setBusy('')
        }
    }

    async function checkNow() {
        setBusy('check')
        try {
            const st = await CheckUpdate()
            if (st.available) {
                Toast.info(`发现新版本 ${st.latest}`)
            } else if (st.error) {
                Toast.warning(st.error)
            } else {
                Toast.success(`已是最新版 ${st.current}`)
            }
        } catch (err) {
            Toast.error(String(err))
        } finally {
            setBusy('')
        }
    }

    async function decideClose(action: 'quit' | 'tray') {
        setAskClose(false)
        try {
            await ConfirmClose({action, remember: rememberClose})
        } catch (err) {
            Toast.error(String(err))
        }
    }

    const gw = dash?.gateway
    const cs = dash?.cursor
    const models = current?.models ?? []
    const compatibleVersion = cs?.compatibleVersion || '3.23.23'

    return (
        <div className="app-shell">
            <header className="topbar">
                <div className="brand">
                    <Typography.Title heading={6} style={{margin: 0}}>
                        CursorGate - Cursor自定义API助手
                    </Typography.Title>
                    <Typography.Text type="tertiary" size="small">
                        适配 Cursor {compatibleVersion}
                    </Typography.Text>
                </div>
                <Space>
                    <Button
                        theme="borderless"
                        type="tertiary"
                        icon={theme === 'light' ? <IconMoon/> : <IconSun/>}
                        onClick={switchTheme}
                    >
                        {theme === 'light' ? '暗色' : '亮色'}
                    </Button>
                    <Button theme="light" type="tertiary" loading={busy === 'restore'} onClick={restore}>
                        一键恢复
                    </Button>
                    <Button theme="solid" type="primary" loading={busy === 'apply'} onClick={apply}>
                        同步到 Cursor
                    </Button>
                </Space>
            </header>

            <div className="body">
                <aside className="sidebar">
                    <div className="sidebar-head">
                        <Typography.Text type="tertiary" size="small">API</Typography.Text>
                        <Button
                            size="small"
                            theme="borderless"
                            type="tertiary"
                            icon={<IconPlus/>}
                            onClick={startNew}
                        >
                            添加
                        </Button>
                    </div>
                    <div className="provider-list">
                        {providers.length === 0 && (
                            <Typography.Text type="tertiary" className="empty">
                                还没有 API。右侧填一条，保存即可。
                            </Typography.Text>
                        )}
                        {providers.map(p => (
                            <button
                                key={p.id}
                                type="button"
                                className={'provider-item' + (p.id === selected ? ' active' : '')}
                                onClick={() => pick(p)}
                            >
                                <div className="provider-name">
                                    <span className={'dot ' + (p.online ? 'on' : 'off')}/>
                                    <Typography.Text strong ellipsis={{showTooltip: true}}>
                                        {p.name}
                                    </Typography.Text>
                                </div>
                                <Space wrap spacing={4} className="provider-meta">
                                    <Typography.Text type="tertiary" size="small">
                                        {p.online ? '已连接' : '未连接'} · {p.enabled ? '启用' : '停用'} · {p.models?.length ?? 0}
                                    </Typography.Text>
                                </Space>
                            </button>
                        ))}
                    </div>
                </aside>

                <main className="main">
                    <section className="editor">
                        <Typography.Text type="tertiary" size="small" className="section-title">
                            {form.id ? '编辑 API' : '新建 API'}
                        </Typography.Text>
                        <div className="form-grid">
                            <div className="row-2">
                                <label className="field">
                                    <Typography.Text type="tertiary" size="small">名称</Typography.Text>
                                    <Input
                                        placeholder="例如 WorkBuddy"
                                        value={form.name}
                                        showClear
                                        onChange={v => setForm({...form, name: v})}
                                    />
                                </label>
                                <label className="field">
                                    <Typography.Text type="tertiary" size="small">Base URL</Typography.Text>
                                    <Input
                                        placeholder="http://127.0.0.1:8899/v1"
                                        value={form.baseUrl}
                                        showClear
                                        onChange={v => setForm({...form, baseUrl: v})}
                                    />
                                </label>
                            </div>
                            <label className="field">
                                <Typography.Text type="tertiary" size="small">API Key</Typography.Text>
                                <Input
                                    mode="password"
                                    placeholder={current?.hasKey ? '密钥已保存，留空则不改' : 'API Key'}
                                    value={form.apiKey}
                                    onChange={v => setForm({...form, apiKey: v})}
                                />
                            </label>
                            <div className="row-actions">
                                <Space>
                                    <Switch
                                        checked={form.enabled}
                                        onChange={v => setForm({...form, enabled: v})}
                                    />
                                    <Typography.Text>启用</Typography.Text>
                                </Space>
                                <Space wrap>
                                    <Button theme="solid" type="primary" loading={busy === 'save'} onClick={save}>
                                        保存
                                    </Button>
                                    <Button theme="light" loading={busy === 'refresh'} onClick={refresh}>
                                        拉取模型
                                    </Button>
                                    {form.id ? (
                                        <Button theme="light" type="danger" loading={busy === 'del'} onClick={remove}>
                                            删除
                                        </Button>
                                    ) : null}
                                    {current ? (
                                        <Button
                                            theme="borderless"
                                            type="tertiary"
                                            onClick={() => toggle(current, !current.enabled)}
                                        >
                                            {current.enabled ? '停用' : '启用'}
                                        </Button>
                                    ) : null}
                                </Space>
                            </div>
                            <label className="field">
                                <Typography.Text type="tertiary" size="small">
                                    {current ? `${current.name} 的模型（${models.length}）` : '模型'}
                                </Typography.Text>
                                <Select
                                    placeholder={models.length ? '查看已拉取的模型' : '还没有模型，保存后点「拉取模型」'}
                                    optionList={models.map(id => ({label: id, value: id}))}
                                    filter
                                    showClear
                                    style={{width: '100%'}}
                                    disabled={!models.length}
                                />
                            </label>
                            {current?.onlineErr ? (
                                <Banner type="warning" closeIcon={null} description={'连接失败：' + current.onlineErr}/>
                            ) : null}
                            {current?.lastError ? (
                                <Banner type="danger" closeIcon={null} description={current.lastError}/>
                            ) : null}
                        </div>
                    </section>
                </main>
            </div>

            <footer className="footer">
                <div className="status-cell">
                    <span className={'dot ' + (gw?.running ? 'on' : 'off')}/>
                    <Typography.Text type="tertiary" size="small">
                        合流 {gw?.running ? gw.addr : (gw?.error || '未启动')}
                    </Typography.Text>
                </div>
                <div className="status-cell">
                    <span className={'dot ' + (cs?.compatible ? 'on' : 'warn')}/>
                    <Typography.Text type="tertiary" size="small">
                        Cursor {cs?.version || '未知'} {cs?.compatible ? '已适配' : '未适配'}
                    </Typography.Text>
                    <Button theme="borderless" type="tertiary" size="small" onClick={() => OpenCursorDownload()}>
                        下载
                    </Button>
                </div>
                <div className="status-cell">
                    <span className={'dot ' + (cs?.useOpenAIKey ? 'on' : 'off')}/>
                    <Typography.Text type="tertiary" size="small" ellipsis={{showTooltip: true}}>
                        {cs?.openAIBaseUrl || '未读取'}
                    </Typography.Text>
                </div>
                <div className="status-cell">
                    <span className={'dot ' + (cs?.cursorRunning ? 'on' : 'off')}/>
                    <Typography.Text type="tertiary" size="small">
                        {cs?.cursorRunning ? '运行中' : '未运行'}
                    </Typography.Text>
                </div>
                <div className="status-cell">
                    <span className={'dot ' + (appVer.available ? 'warn' : 'on')}/>
                    <Typography.Text type="tertiary" size="small" ellipsis={{showTooltip: true}}>
                        {appVer.available ? `v${appVer.current} → ${appVer.latest}` : `v${appVer.current}`}
                    </Typography.Text>
                    {appVer.available ? (
                        <Button theme="borderless" type="warning" size="small" loading={busy === 'update'} onClick={applyUpdate}>
                            更新
                        </Button>
                    ) : (
                        <Button theme="borderless" type="tertiary" size="small" loading={busy === 'check'} onClick={checkNow}>
                            检查
                        </Button>
                    )}
                    <Button theme="borderless" type="tertiary" size="small" onClick={() => OpenReleasePage()}>
                        发布页
                    </Button>
                </div>
            </footer>

            <Modal
                title="关闭窗口"
                visible={askClose}
                onCancel={() => setAskClose(false)}
                footer={
                    <Space>
                        <Button onClick={() => decideClose('quit')}>关闭窗口</Button>
                        <Button theme="solid" type="primary" onClick={() => decideClose('tray')}>缩小到托盘</Button>
                    </Space>
                }
            >
                <Typography.Paragraph>
                    关闭主窗口后，合流仍会在托盘继续运行。也可以直接退出。
                </Typography.Paragraph>
                <Checkbox checked={rememberClose} onChange={e => setRememberClose(!!e.target.checked)}>
                    记住我的选择
                </Checkbox>
            </Modal>
        </div>
    )
}

export default App
