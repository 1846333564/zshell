# wiShell

wiShell 是 Windows 桌面 SSH/SFTP 工具，使用 Go、Wails/WebView2、Vue、xterm.js、Monaco Editor 和基于 SSH 的 SFTP 实现。

## 当前版本

- 当前本地版本：`0.4.2`
- 版本来源：根目录 `VERSION`
- 本地打包产物：`release/wiShell.0.4.2.exe`
- 本版本会通过 `v0.4.2` tag 发布 GitHub Release，标题为“完善文件路径多级联动补全”。

## 文件拆分规则

- 后端 Go 业务文件最好不超过 300 行，禁止超过 500 行。
- 前端 Vue、JS、CSS 业务文件最好不超过 500 行，禁止超过 800 行。
- `package-lock.json` 这类生成文件不按人工拆分规则处理。

## 本次重构

- 日志系统写入 `%AppData%\wiShell\log`，每小时一个 `wiShell-YYYYMMDD-HH.log` 文件，应用启动时清理 24 小时以前的日志。
- HTTP/API 错误、流式 SFTP/更新错误、终端 WebSocket/SSH 错误、HTTP panic 和入口 panic 都会记录错误位置与原始错误原因。
- SFTP 批量上传会先批量创建目录，再对大量小文件做有界并发上传；本地 API 长请求不再使用 30 秒响应写超时。
- 后端 `httpapi`、`sftpsvc`、`updatesvc` 已按职责拆成多个同包文件，保留原函数签名和路由行为。
- 前端 `App.vue` 的连接、更新和 UI 偏好逻辑已拆到 `src/composables/app`。
- 应用更新弹窗支持在下载和校验阶段停止更新，停止后会取消流式请求并阻止进入替换重启流程。
- 文件管理器降低后台目录预加载压力，文本编辑会立即创建窗口并初始化 Monaco，在远程下载期间直接显示已经到达的内容，并修复打开进度层遗漏绑定导致目录树和编辑窗口渲染中断的问题。
- 文件管理器拆分目录/文件右键菜单，支持精确右键目标、空白父目录菜单、原位重命名、整目录复制/剪切/下载、条件粘贴与上传；左侧目录树每个目录独占一行，以 90 度空心折线图标表示状态并由整行点击展开/折叠，右侧千级列表和左树使用固定行高虚拟滚动。
- 文件路径输入会复用递归预加载缓存中的父目录索引，仅在实际输入且存在匹配时显示最多 10 个直接子目录候选；Tab 支持唯一目录、公共前缀、精确目录与唯一更长候选的多级联动补全，并同步导航文件树和文件列表，Enter 可优先确认精确目录。
- 在线编辑通过原始 UTF-8 响应和 XHR 原生进度事件下载内容，增量文本先进入非响应式缓冲区，最多每 1 秒合并后以新文本节点追加到独立只读预览，完整读取后再切换到 Monaco 并解锁编辑；编辑窗口状态从创建起保持 Vue 响应式，进度与内容更新不依赖鼠标点击。关闭编辑窗口会同时取消浏览器请求和服务器 SFTP 读取，避免后台继续无用下载。
- UI 主题系统支持 wiShell 默认、Dracula、Nord、Tokyo Night、Catppuccin Mocha、Gruvbox Dark、One Dark、Solarized Dark 和自定义颜色。
- WebView2 默认开启 GPU 硬件加速，顶部 `UI管理` 可持久化切换并在重启后生效。
- 前端 `FileManager.vue` 保留模板，setup 逻辑拆到 `src/components/useFileManager.js`。
- 全局样式从 `src/style.css` 拆到 `src/styles` 分包文件。

## 验证

```powershell
cd frontend
npm run build

cd ..
powershell -NoProfile -ExecutionPolicy Bypass -File .\build-windows.ps1 -SkipGoTests
```

构建后只启动最终 `release\wiShell.<版本号>.exe` 并确认本地 `/api/health` 返回 `ok`；不进行图像验收，也不运行会生成 `.test.exe` 的 Go 测试。
