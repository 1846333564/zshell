# zShell

zShell 是 Windows 桌面 SSH/SFTP 工具，使用 Go、Wails/WebView2、Vue、xterm.js、Monaco Editor 和基于 SSH 的 SFTP 实现。

## 当前版本

- 当前本地版本：`0.3.8`
- 版本来源：根目录 `VERSION`
- 本地打包产物：`release/zshell.0.3.8.exe`
- 本版本会通过 `v0.3.8` tag 发布 GitHub Release。

## 文件拆分规则

- 后端 Go 业务文件最好不超过 300 行，禁止超过 500 行。
- 前端 Vue、JS、CSS 业务文件最好不超过 500 行，禁止超过 800 行。
- `package-lock.json` 这类生成文件不按人工拆分规则处理。

## 本次重构

- 日志系统写入 `%AppData%\zShell\log`，每小时一个 `zshell-YYYYMMDD-HH.log` 文件，应用启动时清理 24 小时以前的日志。
- HTTP/API 错误、流式 SFTP/更新错误、终端 WebSocket/SSH 错误、HTTP panic 和入口 panic 都会记录错误位置与原始错误原因。
- SFTP 批量上传会先批量创建目录，再对大量小文件做有界并发上传；本地 API 长请求不再使用 30 秒响应写超时。
- 后端 `httpapi`、`sftpsvc`、`updatesvc` 已按职责拆成多个同包文件，保留原函数签名和路由行为。
- 前端 `App.vue` 的连接、更新和 UI 偏好逻辑已拆到 `src/composables/app`。
- UI 主题系统支持 zShell 默认、Dracula、Nord、Tokyo Night、Catppuccin Mocha、Gruvbox Dark、One Dark、Solarized Dark 和自定义颜色。
- 前端 `FileManager.vue` 保留模板，setup 逻辑拆到 `src/components/useFileManager.js`。
- 全局样式从 `src/style.css` 拆到 `src/styles` 分包文件。

## 验证

```powershell
cd frontend
npm run build

cd ..\backend
go test ./...

cd ..
powershell -NoProfile -ExecutionPolicy Bypass -File .\build-windows.ps1
```
