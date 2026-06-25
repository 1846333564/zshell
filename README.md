# zShell

zShell 是 Windows 桌面 SSH/SFTP 工具，使用 Go、Wails/WebView2、Vue、xterm.js、Monaco Editor 和基于 SSH 的 SFTP 实现。

## 当前版本

- 当前本地版本：`0.3.5`
- 版本来源：根目录 `VERSION`
- 本地打包产物：`release/zshell.0.3.5.exe`
- 本版本只做本地 commit 和本地打包，不发布 GitHub Release。

## 文件拆分规则

- 后端 Go 业务文件最好不超过 300 行，禁止超过 500 行。
- 前端 Vue、JS、CSS 业务文件最好不超过 500 行，禁止超过 800 行。
- `package-lock.json` 这类生成文件不按人工拆分规则处理。

## 本次重构

- 后端 `httpapi`、`sftpsvc`、`updatesvc` 已按职责拆成多个同包文件，保留原函数签名和路由行为。
- 前端 `App.vue` 的连接、更新和 UI 偏好逻辑已拆到 `src/composables/app`。
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
