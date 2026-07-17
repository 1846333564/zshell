# 后端模块状态

## 范围

Go 后端负责 Wails 应用启动、本地 API 服务、动态端口绑定和 release 构建集成。

## 重要文件

- `main.go`：Wails 桌面入口。
- `cmd/wiShell/main.go`：保留的 HTTP-only 开发入口。
- `wails.json`：Wails 项目配置。
- `go.mod`：后端依赖。
- `internal/logsvc`：应用日志初始化、按小时轮转和启动清理。
- `../VERSION`：当前版本号来源。
- `../build-windows.ps1`：release 构建入口。

## 当前状态

后端会先初始化日志系统，再在动态高端口启动本地 API/WebSocket 服务，并通过 Wails 加载前端资源。本地 API 使用请求头读取超时和空闲超时，但不为长时间 SFTP 上传、下载、更新等流式响应设置 30 秒写超时。日志写入 `%AppData%\wiShell\log`，按小时轮转并在启动时清理 24 小时以前的日志。Wails 使用无边框 Windows 窗口，在创建 WebView2 前读取加密 UI 偏好，默认开启 GPU 硬件加速，让 Vue、xterm 和 Monaco 通过合成器流畅绘制；用户可关闭该设置并在下次启动生效。后端同时保留 WebView 右键事件，使 Vue 可以渲染自定义文件管理器菜单并屏蔽非文件区菜单。保存的连接配置在 API 服务启动时加载进内存，供终端、SFTP 和监控流程使用；应用改名后会从旧 `%AppData%\zShell` / `%AppData%\zshell` 配置文件迁移保存连接，避免升级后连接列表变空。

release 构建脚本会检查 npm、Go 和 Wails 原生命令退出码，读取 `VERSION`，通过 ldflags 注入运行时版本号，并把最终 exe 复制到项目 `release` 文件夹，文件名为 `wiShell.<版本号>.exe`。脚本只覆盖当前版本对应的 exe，不自动删除旧版本本地 release 包。

## 已知工作

保存密码编辑、终端登录、SFTP 导航和自更新仍需要结合用户真实服务器与真实 GitHub Release 进行验证。
