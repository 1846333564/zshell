# WebSocket 终端模块状态

## 范围

xterm.js 前端和 SSH PTY shell 之间的 WebSocket 桥接。

## 重要文件

- `terminal_handler.go`
- `protocol.go`
- `terminal_handler_test.go`

## 当前状态

支持原始输入、输出流、resize 消息、协议层 ping/pong、服务端 WebSocket ping 帧、读取 deadline，以及 UTF-8 多字节边界处理。

## 已知工作

WebSocket 必须保持在真实本地 API 服务上；Wails 静态资源服务不支持终端 WebSocket upgrade。若用户仍看到空闲断连，需要先捕获终端关闭前的错误码，用于区分 WebSocket 传输失败和远程 SSH shell 自身退出。
