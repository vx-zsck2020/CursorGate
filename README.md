# CursorGate - Cursor自定义API助手

给 Cursor 管多个自定义 API 的本机小工具。

Cursor 原生只有一个 `openAIBaseUrl`。本工具在本机起合流入口（默认 `127.0.0.1:8900`，占用自动换口），按模型 id 分流到各上游，并把聚合后的模型列表写进 Cursor 的 `state.vscdb`。适配 **Cursor 3.23.23**。

仓库：<https://github.com/vx-zsck2020/CursorGate>

## 能做什么

- 同时管理多条 OpenAI 兼容 API（名称、Base URL、密钥用 Windows DPAPI 加密）
- 拉取并合流模型列表，同步到 Cursor 选择器
- 点「同步到 Cursor」会自动退出并重启 Cursor
- 右上角「一键恢复」还原最近状态库备份，并撤回官方模型刷新短路
- 系统托盘：打开界面 / 退出；关闭窗口可缩到托盘并记住选择
- 主题切换、管理员启动、客户区 875×525（小屏按工作区收缩）
- 启动后自动检测 GitHub Release；页脚可检查 / 在线更新

## 怎么用

1. 下载 [Releases](https://github.com/vx-zsck2020/CursorGate/releases/latest) 里的 `CursorGate.exe`（首次会要管理员权限）
2. 添加各 OpenAI 兼容 API，点「拉取模型」
3. 点「同步到 Cursor」
4. 在 Cursor 里直接切模型开会话

合流由本工具守护。Cursor BYOK 指向实际合流地址，例如 `http://127.0.0.1:8900/v1`。

## 在线更新

- 启动约 1 秒后自动查 GitHub `releases/latest`
- 没有 Release 时回退读仓库 `latest.json`
- 发现新版本：页脚显示 `当前 → 最新`，点「更新」下载 `CursorGate.exe`，校验 SHA256 后替换并重启
- 发布约定：tag `v主.次.修订`，附件必须包含 `CursorGate.exe`，建议同时放 `CursorGate.exe.sha256`

## 技术栈

Wails v2 + React 18 + `@douyinfe/semi-ui`

## 开发

```bash
go test ./...
cd frontend && npm run build
wails build
```

本机 22 端口打不到 GitHub 时，用 SSH 443：

```bash
git remote add origin ssh://git@ssh.github.com:443/vx-zsck2020/CursorGate.git
```

打 tag 会触发 Actions 编译并发布 Release：

```bash
git tag v1.0.0
git push origin v1.0.0
```

## 不会做的事

- 不改 `workbuddy-proxy.py`
- 不动 named tunnel
- 不重打 Cursor 五处 JS 补丁（刷新短路除外）
