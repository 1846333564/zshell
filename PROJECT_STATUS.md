# wiShell 项目状态

## 项目概览

wiShell 是一个 Windows 桌面 SSH/SFTP 工具，技术栈包括 Go、Wails/WebView2、Vue、xterm.js、Monaco Editor 和基于 SSH 的 SFTP。当前版本从 `VERSION` 文件读取，本次版本为 `0.3.16`，版本号从 `0.0.1` 起步。发布产物输出到项目根目录的 `release` 文件夹，命名格式为 `wiShell.<版本号>.exe`；本地 `release` 历史包不会自动删除。

## 当前架构

- `backend/main.go` 是 Wails 桌面入口，会在 `127.0.0.1` 的动态高端口启动本地 HTTP API/WebSocket 服务，并在原生 WebView2 窗口中打开 Vue 应用。
- `backend/cmd/wiShell/main.go` 是保留的旧本地 HTTP 开发入口，不负责启动浏览器。
- `backend/internal/appinfo` 保存产品名、版本号、公司、开发者、内测状态和 GitHub Release 仓库信息。
- `backend/internal/httpapi` 暴露连接、SSH、SFTP、远程文本编辑、更新检查、应用更新、跨服务器传输、归档下载和监控接口。
- `backend/internal/updatesvc` 通过 GitHub Release 检查新版本，下载 `wiShell.<版本号>.exe`，并用独立 PowerShell helper 替换当前运行的 exe 后重启。
- `backend/internal/configstore` 使用 Windows DPAPI 在当前用户配置目录中加密保存连接配置。
- `frontend/src/App.vue` 管理双栏桌面壳：左侧监控面板、右侧连接标签、终端和文件区域，并提供“关于 wiShell”和“检查更新”弹窗。
- `build-windows.ps1` 是 release 构建入口，会失败即停地执行 npm、Go 和 Wails 命令，读取 `VERSION`，并输出当前版本 exe 到 `release` 文件夹，不清理旧版本 exe。
- `.github/workflows/release.yml` 用 GitHub Actions 在 tag 或手动触发时构建 Windows exe，并创建或更新 GitHub Release 资产；`.github/release-names.json` 可为指定版本配置 Release 标题，本次 `0.3.16` 已配置“优化在线编辑流式渲染稳定性”标题。
- 后端大文件已按同包功能拆分，HTTP API、SFTP 和更新服务的 Go 源文件都控制在 300 行以内；前端大组件和样式已拆分为组合函数与 CSS 分包，`useFileManager.js` 已恢复为可读多行源码并拆出 `fileManagerUtils.js` 路径/格式化/常量工具，后续可继续按目录树、预加载、编辑器窗口和文件动作拆分。
- `backend/internal/logsvc` 在启动时初始化日志系统，日志写入当前用户配置目录 `%AppData%\wiShell\log`，按小时生成 `wiShell-YYYYMMDD-HH.log`，并在每次启动时清理 24 小时以前的日志文件。

## 已实现

- 密码和当前 Windows 用户 `~/.ssh/id_rsa` SSH 认证；每次连接测试成功后会读取并存储服务器硬件信息，包括 CPU 硬件线程数、核心数、CPU 型号、内存大小和读取时间。
- WebSocket 交互式 PTY 终端。
- SFTP 浏览、上传、下载、归档下载、远程文本读写、远程复制/移动和选中项强制删除；常用 SFTP 操作复用共享 SSH 客户端减少重复握手，共享 SSH 活性探测有短超时并在短时间内复用最近一次探测结果，避免目录浏览、监控和文本读取频繁触发 keepalive；在线编辑创建 SFTP 客户端失败时会丢弃旧连接后重试一次；上传会先批量创建远程目录，再对大量小文件执行有界并发 SFTP 写入并持续回传进度，在线编辑读取进度由后端按真实 SFTP 字节流式回传且支持 256 MiB 文本文件，同服务器复制/移动走远端 `cp`/`mv` 快路径，同服务器删除走远端 `rm -rf` 快路径，跨服务器传输保留 SFTP 流式复制；复制粘贴会避开源路径和已有同名目标，避免表现成剪切或覆盖。
- Wails Windows 可执行文件打包。
- 基于 `VERSION` 的版本号管理；默认后续版本只递增最后一位。
- GitHub Release 更新检查和自更新链路，包含 API 限流 fallback、下载重试、校验、手动下载入口，以及应用更新时的流式阶段进度、下载字节进度、重试日志和停止更新能力；停止会取消流式更新请求，后端收到上下文取消后清理临时下载文件且不会进入替换重启。
- Linux 监控快照 API 和左侧监控 UI；左侧监控 1 秒自动刷新，展示服务器基础参数、服务器时间、CPU/内存/磁盘、进程、分区和上下行网速折线图，网速图只显示峰值并带左侧纵轴刻度。
- 10000 以上动态后端端口。
- 后端管理的保存连接配置增删改查，配置使用 Windows DPAPI 加密落盘。
- 前端保存连接编辑，支持前端模式、后端模式、运维模式；文件管理器会按模式分别默认打开 `/var`、`/opt`、`/`。
- 连接标签只显示连接名。
- 文件管理器路径导航：固定根路径 `/`、解析后的 home 路径如 `/root`、树节点只显示 basename、目录树和右侧列表共享目录缓存与删除失效逻辑，目录树使用与右侧列表一致的分割线和选中条、路径输入支持全局已知目录的 Tab 唯一补全、路径历史按内存访问频次和最近访问排序并默认滚到底部，右侧居中折叠按钮、完整右键菜单动作、选中项删除确认、可调整文件列表列宽、列头点击排序动画和升降序箭头；已打开目录会先使用内存缓存秒开，再后台刷新真实目录内容，目录树索引按小批量写入，目录预加载在当前目录加载完成后立即后台启动，每批最多缓存 10 个未加载目录，并会基于已缓存目录继续递归推进，手动刷新和 F5 会重新索引所有已打开目录。
- 文件管理器在线文本编辑：双击或右键打开普通浮动窗口，支持多个文件窗口同时编辑、拖动、最小化、最大化、`Ctrl+S` 保存，关闭脏内容时提示保存、保存并关闭、不保存并关闭或取消；编辑区已重构为 Monaco Editor，支持代码高亮、Tab 缩进、`Ctrl+F` 搜索、匹配高亮和替换能力；打开文件时窗口和编辑器会立即出现，后端先按 32 KiB 分块流式返回文本内容，前端收到首块后立即追加显示，后台继续多批次读取剩余内容，完整读取前编辑区保持只读且保存禁用；流式读取不可用时会自动回退到普通文本读取，编辑器相关模板绑定已显式接入，避免读取中或窗口交互时文件管理器渲染中断。
- 在线编辑流式打开路径不再在 `apiClient` 中默认收集完整响应，也不再用每次 chunk 更新完整 `v-model` 字符串；文件管理器把文本按约 32 KiB 或一个 animation frame 合批写入 `contentChunks`，`RemoteCodeEditor` 通过 `appendVersion` 每帧最多消费一块并差量追加到 Monaco 模型或降级 textarea。远程读取完成后才一次性合成 `content` / `originalContent` 作为保存基线；如果编辑器渲染队列还没追上，会保持只读并禁用保存，避免用户编辑时仍有后台 chunk 写入。
- 文件和终端右键菜单渲染在视口层，避免 UI 缩放造成坐标偏移；文件右键菜单点击其他位置会关闭。
- 文件选择器或拖放上传，显示后端真实 SFTP 写入总进度、单文件进度、上传速度，上传面板可折叠并保留最近一次上传记录。
- 终端聚焦时 `Ctrl +` / `Ctrl -` 调整字体并持久化到加密配置文件；非终端 UI 缩放也会持久化，终端区域会抵消全局 UI zoom，避免 xterm.js 鼠标选择坐标相对光标左上偏移，同时终端自身保持正常 `100%` flex 尺寸，避免 UI 缩放二次参与布局导致右侧或下方空白；终端支持 `Ctrl+Shift+C` / `Ctrl+Shift+V` 剪贴板快捷键。
- Monaco Editor 通过动态 import 和 Web Worker 懒加载接入，应用首屏不静态加载 Monaco；启动后会延后到更空闲的阶段后台预热，在线编辑器窗口打开后即可挂载 Monaco，远程内容按 32 KiB 分块并经前端帧泵追加到已有编辑器模型，首次打开会复用同一个加载 Promise。
- 交互式终端使用 SSH keepalive 和服务端 WebSocket ping/pong，降低空闲或后台断连概率。
- Wails 窗口为无边框窗口，WebView2 GPU 已禁用以降低启动高负载阶段的渲染进程崩溃风险，并带自定义 wiShell 顶栏、`配置管理` 菜单、可用的 `UI管理 -> 主题设置`、自定义窗口控制和应用风格滚动条；默认主题保持原有 wiShell 配色，并新增 Dracula、Nord、Tokyo Night、Catppuccin Mocha、Gruvbox Dark、One Dark、Solarized Dark 和自定义颜色主题。
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
