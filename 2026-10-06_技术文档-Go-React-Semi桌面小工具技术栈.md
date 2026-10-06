# Go + React + Semi Design 桌面小工具技术栈

> 分析日期：2026-10-06  
> 文档性质：选型拍板 + 落地约束  
> 目标形态：简单操作小工具（无表格、低密度）  
> 状态：**已拍板，未实施**

---

## 0. 先确认我理解的意思

要的不是「再找一个 Go 写的 Vue 组件库」，也不是把丸子 / 小蜜的 Tauri + Vue 栈搬过来。

**已经拍板的是这一条：**

1. 桌面壳用 **Wails v2**（Go 后端 + 系统 WebView）。
2. 窗口内 UI 用 **React 18 + TypeScript + Vite**。
3. 组件库用 Semi 官方包 **`@douyinfe/semi-ui`**（React &lt; 19）。
4. 产品是操作小工具：无表格、无侧栏、低密度。

**明确不做：**

- 不走 Vue（官方 Semi 没有 Vue 实现）。
- 不用非官方 Vue 移植。
- 不用 Wails v3（仍是 Alpha）。
- 不上 React 19 / `@douyinfe/semi-ui-19`（官方 `react-ts` 模板对 React 18）。
- 不把 Semi 的中后台骨架（Layout / Navigation / Table / 重 Form）搬进小工具。

---

## 1. 执行摘要

Go 生态没有「Go + Vue 一体的桌面组件库」。正确拆法是：

```text
Go 业务  →  Wails v2 壳  →  React 18 窗口  →  Semi 官方组件
```

系统能力（菜单、文件对话框、托盘）走 Wails Runtime。  
窗口内表单、按钮、反馈走 Semi。  
两层不要互相冒充。

当前推荐版本口径：

| 层 | 选择 | 口径 |
|---|---|---|
| 语言 | Go 1.22+ | Wails v2 生产线 |
| 桌面壳 | Wails v2.15 | 官方现网文档版本；生产用 v2 |
| 前端 | React 18 + TS + Vite | `wails init -t react-ts` |
| 组件库 | `@douyinfe/semi-ui` | Semi 2.103 线，官方 React 实现 |
| 图标 | `@douyinfe/semi-icons` | 与 UI 包配套 |
| 产品密度 | 低 | 无表格、无侧栏导航 |

---

## 2. 范围

- **in_scope**：新开的 Go 桌面小工具选型；Wails 壳 + Semi 窗口内组件；低密度布局约束。
- **out_scope**：不改丸子 / 小蜜（仍是 Tauri + Vue）；不实施项目脚手架，除非另下命令；不引入 mux / 云服务 / 多窗口框架。
- **network_profile**：默认本机工具，无强制联网。

---

## 3. 为什么这样拆

### 3.1 不存在「Go + Vue 桌面组件库」

桌面 GUI 在 Go 里是三路，不能混：

| 路线 | 代表 | 和本次需求 |
|---|---|---|
| 纯 Go 自绘 | Fyne / Gio | 没有 Vue，也没有 Semi |
| 原生绑定 | Walk 等 | 没有 Vue |
| WebView 壳 | **Wails** | Go 后端 + 任意前端；官方有 Vue / React 模板，**不含组件库** |

所以「Go + Vue + 优质桌面组件库」实际是「Wails + 某套 Vue UI」。  
一旦坚持 Semi Design，Vue 这条路就断了：Semi 官方只交付 React Adapter。

### 3.2 Vue 移植不作为生产依赖

存在非官方复刻（例如 `aifuxi/semi-ui-vue`，基线 Semi v2.102.0）：

- 个人维护，大量 AI 辅助
- README 写明**不是官方**、稳定版验收未完成、可能与官方行为不一致
- 小工具不该把视觉和交互压在未验收的复刻上

结论：要 Semi，就用官方 React 包，前端改 React。

### 3.3 组件库为什么是 Semi，不是 Naive / TDesign

此前若走 Vue，工具型低密度更适合 Naive UI。  
现已改口要 Semi 的设计语言，则：

- Semi 官方包完整、Token / 暗色 / 文档都在 React 上
- TDesign Vue Next 虽自称 desktop application，但是 Vue 库，和 Semi 不是同一套视觉
- Element Plus 中后台味重，不适合低密度小工具

---

## 4. 选定技术栈

### 4.1 分层

```text
┌─────────────────────────────────────────┐
│  Go 业务（文件、系统调用、核心逻辑）      │
└─────────────────┬───────────────────────┘
                  │ Wails bind / events
┌─────────────────▼───────────────────────┐
│  Wails v2 壳                             │
│  WebView2 · 原生菜单 · 对话框 · 托盘     │
└─────────────────┬───────────────────────┘
                  │ JS / TS 绑定
┌─────────────────▼───────────────────────┐
│  React 18 + Vite + TypeScript            │
│  @douyinfe/semi-ui + semi-icons          │
└─────────────────────────────────────────┘
```

### 4.2 包与模板

起手：

```bash
wails init -n <工具名> -t react-ts
cd <工具名>/frontend
npm i @douyinfe/semi-ui @douyinfe/semi-icons
```

说明：

- 官方模板是空壳，社区没有 Semi 专用 Wails 模板，不需要等。
- Webpack / Vite 下 Semi **零配置按需打包**，直接：

```tsx
import { Button, Toast } from '@douyinfe/semi-ui';
```

- React 19 才换 `@douyinfe/semi-ui-19`。本栈锁 React 18，不要混装。

### 4.3 职责边界

| 能力 | 走哪一层 | 不要做的 |
|---|---|---|
| 打开 / 保存文件 | Wails Runtime 对话框 | 不要用 Semi Upload 冒充系统选文件（除非明确要拖拽区） |
| 应用菜单、托盘 | Wails | 不要用 Semi Navigation 做系统菜单 |
| 窗口内按钮、输入、开关 | Semi | 不要手写一套控件 |
| 应用内确认、轻提示 | Semi `Modal` / `Toast` | 不要用它们冒充系统对话框 |
| 暗色 | Semi Token + 主题切换 | 不要写死 `#xxx` |
| 业务计算、IO | Go | 不要把重逻辑堆在 React |

---

## 5. 小工具 UI 约束（低密度）

窗口结构：

```text
一条顶栏（标题 + 一两个动作）
正文：操作区（输入 + 主按钮）
底部或折叠：状态 / 日志 / 次要说明
```

**用这些就够：**

- `Button` / `Input` / `Select` / `Switch`
- `Space` / `Typography` / `Divider`
- `Toast` / `Banner` / `Progress`
- 次要信息：`Collapse` 或一张 `Card`

**不要上：**

- `Layout` 侧栏、`Navigation`、`Tabs` 当主结构
- `Table`、`Form` 重型中后台骨架
- 仪表盘卡片墙、装饰渐变、圆头像

其它约束：

- 主按钮少量 `theme="solid"` + `type="primary"`，其余 `light` / `tertiary`
- 链接型文字走 `Typography` 的 `link`，不要用 Button 冒充
- `Button` 必须从 `@douyinfe/semi-ui` 导入；从 `.../button/button` 拿的是裸组件，`icon` 会失效
- 间距按小工具留白，不要搬中后台 8px 栅格
- 颜色只用 `--semi-color-*` Token，保证暗色能切

---

## 6. 否决项（再讨论也不改，除非另拍板）

| 选项 | 否决理由 |
|---|---|
| 官方 Semi + Vue | 官方只有 React Adapter |
| `semi-ui-vue` 等移植 | 非官方、未验收 |
| Wails v3 | Alpha，模板自己也写生产用 v2 |
| `@douyinfe/semi-ui-19` | 多一个适配层；模板仍是 React 18 |
| Next.js / Remix Wails 模板 | 小工具不需要 SSR |
| Fyne / Walk / Gio | 没有 Semi，也没有 React |
| 把丸子栈（Tauri + Vue）复用过来 | 另一条产品线，互不影响 |

---

## 7. 与现有项目的关系

| 项目 | 栈 | 关系 |
|---|---|---|
| 丸子配音 / MaruAudio | Tauri + Vue | 不动 |
| 小蜜 / Hermes_Agent | Tauri + Vue | 不动 |
| 本小工具 | **Wails v2 + React 18 + Semi** | 新开，独立仓库 |

不要为了对齐旧项目把 Semi 硬接到 Vue 上。

---

## 8. 落地时的最小清单（实施阶段再用）

1. `wails init -n <工具名> -t react-ts`
2. 前端安装 `@douyinfe/semi-ui`、`@douyinfe/semi-icons`
3. 根节点包一层暗色 / 主题切换（Token，不写死色）
4. 页面只放操作区，不上 Layout
5. 文件选择走 Wails 对话框
6. 自测：`wails dev` 能开窗；主按钮能调通一个 Go 方法；暗色能切

本文只定栈，不创建仓库。

---

## 9. 当前结论

一句话：

> **新开的操作小工具定为 Wails v2 + React 18 + 官方 `@douyinfe/semi-ui`；Vue 与非官方移植不做；系统壳和窗口内组件分层；低密度、无表格。**
