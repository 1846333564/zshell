# wiShell 项目状态

## 项目概览

wiShell 是一个 Windows 桌面 SSH/SFTP 工具，技术栈包括 Go、Wails/WebView2、Vue、xterm.js、Monaco Editor 和基于 SSH 的 SFTP。当前版本从 `VERSION` 文件读取，本次版本为 `0.4.1`，版本号从 `0.0.1` 起步。发布产物输出到项目根目录的 `release` 文件夹，命名格式为 `wiShell.<版本号>.exe`；本地 `release` 历史包不会自动删除。

## 当前架构

- `backend/main.go` 是 Wails 桌面入口，会在 `127.0.0.1` 的动态高端口启动本地 HTTP API/WebSocket 服务，并在原生 WebView2 窗口中打开 Vue 应用。
- `backend/cmd/wiShell/main.go` 是保留的旧本地 HTTP 开发入口，不负责启动浏览器。
- `backend/internal/appinfo` 保存产品名、版本号、公司、开发者、内测状态和 GitHub Release 仓库信息。
- `backend/internal/httpapi` 暴露连接、SSH、SFTP、远程文本编辑、更新检查、应用更新、跨服务器传输、归档下载和监控接口。
- `backend/internal/updatesvc` 通过 GitHub Release 检查新版本，下载 `wiShell.<版本号>.exe`，并用独立 PowerShell helper 替换当前运行的 exe 后重启。
- `backend/internal/configstore` 使用 Windows DPAPI 在当前用户配置目录中加密保存连接配置。
- `frontend/src/App.vue` 管理双栏桌面壳：左侧监控面板、右侧连接标签、终端和文件区域，并提供“关于 wiShell”和“检查更新”弹窗。
- `build-windows.ps1` 是 release 构建入口，会失败即停地执行 npm、Go 和 Wails 命令，读取 `VERSION`，并输出当前版本 exe 到 `release` 文件夹，不清理旧版本 exe。脚本保留默认 Go 测试能力；当前 Codex 与 GitHub Release 流程按用户的冒烟策略传入 `-SkipGoTests`，避免生成 `.test.exe`，只验证最终 exe 可启动且 `/api/health` 返回成功。
- `.github/workflows/release.yml` 用 GitHub Actions 在 tag 或手动触发时构建 Windows exe，对构建出的确切 EXE 执行启动和 `/api/health` 冒烟后创建或更新 GitHub Release 资产；`.github/release-names.json` 可为指定版本配置 Release 标题，本次 `0.4.1` 已配置“巨幅优化文件编辑器性能”标题。
- 后端大文件已按同包功能拆分，HTTP API、SFTP 和更新服务的 Go 源文件都控制在 300 行以内；前端大组件和样式已拆分为组合函数与 CSS 分包，`useFileManager.js` 已恢复为可读多行源码并拆出 `fileManagerUtils.js` 路径/格式化/常量工具，后续可继续按目录树、预加载、编辑器窗口和文件动作拆分。
- `backend/internal/logsvc` 在启动时初始化日志系统，日志写入当前用户配置目录 `%AppData%\wiShell\log`，按小时生成 `wiShell-YYYYMMDD-HH.log`，并在每次启动时清理 24 小时以前的日志文件。

## 已实现

- 密码和当前 Windows 用户 `~/.ssh/id_rsa` SSH 认证；每次连接测试成功后会读取并存储服务器硬件信息，包括 CPU 硬件线程数、核心数、CPU 型号、内存大小和读取时间。
- WebSocket 交互式 PTY 终端。
- SFTP 浏览、上传、下载、归档下载、远程文本读写、远程复制/移动和选中项强制删除；常用 SFTP 操作复用共享 SSH 客户端减少重复握手，共享 SSH 活性探测有短超时并在短时间内复用最近一次探测结果，避免目录浏览、监控和文本读取频繁触发 keepalive；在线编辑创建 SFTP 客户端失败时会丢弃旧连接后重试一次；上传会先批量创建远程目录，再对大量小文件执行有界并发 SFTP 写入并持续回传进度，在线编辑读取进度由后端按真实 SFTP 字节流式回传且支持 256 MiB 文本文件，同服务器复制/移动走远端 `cp`/`mv` 快路径，同服务器删除走远端 `rm -rf` 快路径，跨服务器传输保留 SFTP 流式复制；复制粘贴会避开源路径和已有同名目标，避免表现成剪切或覆盖。
- Wails Windows 可执行文件打包。
- 基于 `VERSION` 的版本号管理；默认后续版本只递增最后一位。
- GitHub Release 更新检查和自更新链路，包含 API 限流 fallback、下载重试、校验、手动下载入口，以及应用更新时的流式阶段进度、下载字节进度、重试日志和停止更新能力；停止会取消流式更新请求，后端收到上下文取消后清理临时下载文件且不会进入替换重启。
- Linux 监控快照 API 和左侧监控 UI；左侧监控 1 秒自动刷新，展示服务器基础参数、服务器时间、CPU/内存/磁盘、进程、分区和上下行网速折线图，网速图只显示峰值并带左侧纵轴刻度。刷新期间保留上一帧服务器时间，不再闪烁“更新中”，并在时间后显示本次监控请求延迟毫秒数。
- 10000 以上动态后端端口。
- 后端管理的保存连接配置增删改查，配置使用 Windows DPAPI 加密落盘。
- 保存连接配置当前写入 `%AppData%\wiShell\connections.dpapi`；如果新配置不存在，或新配置尚未迁移且连接列表为空，会自动读取旧版 `%AppData%\zShell\connections.dpapi` / `%AppData%\zshell\connections.dpapi`，把旧连接复制到新配置并记录迁移标记，避免应用改名后保存连接记录消失。
- 前端保存连接编辑，支持前端模式、后端模式、运维模式；文件管理器会按模式分别默认打开 `/var`、`/opt`、`/`。
- 连接标签只显示连接名。
- 文件管理器路径导航：固定根路径 `/`、解析后的 home 路径如 `/root`，左侧目录树只显示目录且每个目录独占一行，不再压缩唯一连续子目录；左侧使用与 VS Code 同类的 90 度空心折线图标，不再把展开功能放在独立按钮上，整行点击会选中并在展开/折叠间切换，刷新目录时保留用户的折叠状态。目录树和右侧列表使用独立且明确的右键目标规则，右侧空白/表头回退到当前父目录，目录与文件拥有分离菜单，支持刷新、展开、整目录下载、原位重命名、上传、复制、红色剪切/删除、条件粘贴和复制路径；菜单不再显示顶部路径标题，并在渲染后按真实高度适配视口，极矮窗口使用菜单内滚动兜底。目录树与文件列表共享目录缓存及动作后的定向刷新，右侧点击会把左树定位到对应父目录；文件和目录列表使用固定行高虚拟滚动，在千级项目下只渲染可见窗口。路径输入继续支持全局已知目录的 Tab 唯一补全，路径历史按内存访问频次和最近访问排序并默认滚到底部，文件列表支持可调列宽、排序动画，并复用同一个空心折线图标表示升降序；已打开目录会先使用内存缓存秒开，再后台刷新真实目录内容，目录预加载在当前目录加载完成后立即后台启动，每批最多缓存 10 个未加载目录，并会基于已缓存目录继续递归推进，手动刷新和 F5 会重新索引所有已打开目录。
- 文件管理器在线文本编辑：双击或右键打开普通浮动窗口，支持多个文件窗口同时编辑、拖动、最小化、最大化、`Ctrl+S` 保存，关闭脏内容时提示保存、保存并关闭、不保存并关闭或取消；编辑区使用 Monaco Editor，支持代码高亮、Tab 缩进、`Ctrl+F` 搜索、匹配高亮和替换能力。打开文件时窗口和下载进度会立即出现，后端通过 `/api/sftp/file/read/raw` 返回原始 UTF-8 内容和独立的预期文件大小响应头并显式刷新响应块；前端使用 XHR 原生进度事件显示真实下载字节，按 `responseText` 字符偏移取得的新内容先进入非响应式缓冲区，最多每 1 秒合并为一个批次并作为文本节点追加到独立只读预览，避免持续重建越来越大的 textarea，也不在 WebView 主线程逐 32 KiB 解析 NDJSON/Base64。完成后会核对字节长度、设置最终 `content` / `originalContent`，再把全文交给 Monaco、切换文件语言并开放编辑保存。
- 每个编辑窗口拥有独立的非响应式 `AbortController`。真正关闭窗口或卸载文件管理器时会立即中止对应 XHR；HTTP handler 监听请求上下文并关闭 SFTP 文件和会话，使未完成的服务器下载同步停止。取消不会回退成第二次普通读取，也不会显示成打开失败；编辑器保存固定使用窗口创建时的连接 ID，避免连接状态变化后写入错误服务器。
- 文件和终端右键菜单渲染在视口层，避免 UI 缩放造成坐标偏移；文件右键菜单点击其他位置会关闭。
- 文件选择器或拖放上传，显示后端真实 SFTP 写入总进度、单文件进度、上传速度，上传面板可折叠并保留最近一次上传记录。
- 终端聚焦时 `Ctrl +` / `Ctrl -` 调整字体并持久化到加密配置文件；非终端 UI 缩放也会持久化，终端区域会抵消全局 UI zoom，避免 xterm.js 鼠标选择坐标相对光标左上偏移，同时终端自身保持正常 `100%` flex 尺寸，避免 UI 缩放二次参与布局导致右侧或下方空白；终端支持 `Ctrl+Shift+C` / `Ctrl+Shift+V` 剪贴板快捷键。
- Monaco Editor 通过动态 import 和 Web Worker 懒加载接入，应用首屏不静态加载 Monaco；启动后会延后到更空闲的阶段后台预热，远程下载期间由独立只读预览在首秒内显示已到达文本并按约 1 秒批量增长，Monaco 在后台初始化并在下载完成后接收全文、完成布局和语言切换，首次打开会复用同一个加载 Promise。
- 交互式终端使用 SSH keepalive 和服务端 WebSocket ping/pong，降低空闲或后台断连概率。
- Wails 窗口为无边框窗口，WebView2 默认开启 GPU 硬件加速以确保空数据和复杂界面的悬浮、点击、滚动及动画及时合成；顶部 `UI管理` 可持久化切换 GPU 渲染，重启后生效。窗口同时带自定义 wiShell 顶栏、`配置管理` 菜单、主题设置、自定义窗口控制和应用风格滚动条；默认主题保持原有 wiShell 配色，并新增 Dracula、Nord、Tokyo Night、Catppuccin Mocha、Gruvbox Dark、One Dark、Solarized Dark 和自定义颜色主题。
- 未连接首页不渲染左侧监控栏；连接失败信息限制在连接配置面板内换行，避免撑开首页两栏布局。
- 应用启动后会后台静默检查更新；检查失败或没有更新不打扰用户，发现新版本时才弹窗提示。
- 日志系统会记录本地 API 返回的错误、流式上传/读取/更新错误、终端 WebSocket/SSH 错误、HTTP panic、入口 panic 和后台 API 服务异常，日志内容保留错误位置和原始错误原因。

## 当前缺口

- 更完整的真实服务器验证仍依赖用户提供 SSH 目标。
- SFTP 所属用户展示目前使用协议层 UID:GID；若要展示友好用户名，需要额外远程查询。
- 在线编辑当前按文本处理并限制 256 MiB；二进制安全编辑和显式编码选择是后续工作。
- GitHub Release 自更新依赖公开可访问的 Release 资产；私有仓库发布需要额外认证方案。用户当前网络若无法直连 GitHub 资产，仍需要配置可被进程继承的代理或手动下载安装包。

## 必要工作流

以后修改本项目之前，先读本文件和涉及模块下的 `MODULE_STATUS.md`。每次代码修改后递增版本号，运行 `powershell -ExecutionPolicy Bypass -File .\build-windows.ps1`，确保生成 `release\wiShell.<版本号>.exe`，再复制一份到 `D:\` 根目录，并对最终 exe 做基本冒烟验证。不要自动删除 `release` 中的旧 exe。验证通过后提交并推送到 GitHub，并通过推送 `v<版本号>` tag 触发 GitHub Release 发布。
