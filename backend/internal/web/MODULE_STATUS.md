# Web 资源模块状态

## 范围

嵌入前端资源服务，以及运行时后端基础地址注入。

## 重要文件

- `server.go`
- `app/.gitkeep`

## 当前状态

Wails 窗口通过 `web.HandlerWithConfig` 加载 Vue 资源，并注入 `window.__WISHELL_BACKEND_BASE__`，供 HTTP 和 WebSocket 调用使用。

## 已知工作

不要通过 Wails 静态资源服务承载终端 WebSocket。
