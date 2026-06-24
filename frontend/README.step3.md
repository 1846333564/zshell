# zShell - 第 3 步

## 已实现内容

- Vue 3 + Vite 前端脚手架。
- 集成 xterm.js 终端组件。
- 实现连接表单。
- 接入连接创建和测试 HTTP API。
- 接入 `/ws/terminal` WebSocket。
- 打通终端输入/输出端到端链路。

## 启动顺序

1. 在一个终端启动后端：

```powershell
cd backend
go run ./cmd/zshell
```

2. 在另一个终端启动前端：

```powershell
cd frontend
npm install
npm run dev
```

3. 打开：

```text
http://127.0.0.1:5173
```

## 使用方式

1. 填写 host、port、username、password。
2. 点击“测试并连接”。
3. 连接后，在终端中输入命令并按 Enter。
4. 输出会流式返回到 xterm 面板。

## 说明

第 3 步重点是 xterm 和 WebSocket I/O。`top`、`vim` 等交互式 PTY 行为在第 4 步完成。
