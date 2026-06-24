# zShell 项目状态

## 项目概览

zShell 是一个 Windows 桌面 SSH/SFTP 工具，技术栈包括 Go、Wails/WebView2、Vue、xterm.js 和基于 SSH 的 SFTP。当前版本从 `VERSION` 文件读取，本次版本为 `0.0.3`，版本号从 `0.0.1` 起步。发布产物输出到项目根目录的 `release` 文件夹，命名格式为 `zshell.<版本号>.exe`。

## 当前架构

- `backend/main.go` 是 Wails 桌面入口，会在 `127.0.0.1` 的动态高端口启动本地 HTTP API/WebSocket 服务，并在原生 WebView2 窗口中打开 Vue 应用。
- `backend/cmd/zshell/main.go` 是保留的旧本地 HTTP 开发入口，不负责启动浏览器。
- `backend/internal/appinfo` 保存产品名、版本号、公司、开发者、内测状态和 GitHub Release 仓库信息。
- `backend/internal/httpapi` 暴露连接、SSH、SFTP、远程文本编辑、更新检查、应用更新、跨服务器传输、归档下载和监控接口。
- `backend/internal/updatesvc` 通过 GitHub Release 检查新版本，下载 `zshell.<版本号>.exe`，并用独立 PowerShell helper 替换当前运行的 exe 后重启。
- `backend/internal/configstore` 使用 Windows DPAPI 在当前用户配置目录中加密保存连接配置。
- `frontend/src/App.vue` 管理双栏桌面壳：左侧监控面板、右侧连接标签、终端和文件区域，并提供“关于 zShell”和“检查更新”弹窗。
- `build-windows.ps1` 是 release 构建入口，会失败即停地执行 npm、Go 和 Wails 命令，读取 `VERSION`，并只保留 `release` 文件夹中的一个 exe。
- `.github/workflows/release.yml` 用 GitHub Actions 在 tag 或手动触发时构建 Windows exe，并创建或更新 GitHub Release 资产。

## 已实现

- 密码和当前 Windows 用户 `~/.ssh/id_rsa` SSH 认证。
- WebSocket 交互式 PTY 终端。
- SFTP 浏览、上传、下载、归档下载、远程文本读写和远程复制/移动。
- Wails Windows 可执行文件打包。
- 基于 `VERSION` 的版本号管理；默认后续版本只递增最后一位。
- GitHub Release 更新检查和自更新链路，包含 API 限流 fallback、下载重试、校验和手动下载入口。
- Linux 监控快照 API 和左侧监控 UI。
- 10000 以上动态后端端口。
- 后端管理的保存连接配置增删改查，配置使用 Windows DPAPI 加密落盘。
- 前端保存连接编辑。
- 连接标签只显示连接名。
- 文件管理器路径导航：固定根路径 `/`、解析后的 home 路径如 `/root`、树节点只显示 basename、打开状态用颜色标记、右侧居中折叠按钮、完整右键菜单动作和可调整文件列表列宽。
- 文件管理器在线文本编辑：双击或右键打开独立编辑窗口，`Ctrl+S` 或保存按钮上传替换远程文件，关闭脏内容时提示保存、不保存或取消。
- 文件和终端右键菜单渲染在视口层，避免 UI 缩放造成坐标偏移；文件右键菜单点击其他位置会关闭。
- 文件选择器或拖放上传，显示总进度、单文件进度、上传速度，并在完成后自动关闭紧凑进度面板。
- 终端聚焦时 `Ctrl +` / `Ctrl -` 调整字体，非终端 UI 缩放持久化，终端支持 `Ctrl+Shift+C` / `Ctrl+Shift+V` 剪贴板快捷键。
- 交互式终端使用 SSH keepalive 和服务端 WebSocket ping/pong，降低空闲或后台断连概率。
- Wails 窗口为无边框窗口，带自定义 zShell 顶栏、占位的 `配置管理` 和 `UI管理` 菜单、自定义窗口控制和应用风格滚动条。
- 未连接首页不渲染左侧监控栏；连接失败信息限制在连接配置面板内换行，避免撑开首页两栏布局。

## 当前缺口

- 更完整的真实服务器验证仍依赖用户提供 SSH 目标。
- SFTP 所属用户展示目前使用协议层 UID:GID；若要展示友好用户名，需要额外远程查询。
- 在线编辑当前按文本处理并限制 10 MB；二进制安全编辑和显式编码选择是后续工作。
- GitHub Release 自更新依赖公开可访问的 Release 资产；私有仓库发布需要额外认证方案。用户当前网络若无法直连 GitHub 资产，仍需要配置可被进程继承的代理或手动下载安装包。

## 必要工作流

以后修改本项目之前，先读本文件和涉及模块下的 `MODULE_STATUS.md`。每次代码修改后运行 `powershell -ExecutionPolicy Bypass -File .\build-windows.ps1`，确保生成 `release\zshell.<版本号>.exe`，并对最终 exe 做基本冒烟验证。
