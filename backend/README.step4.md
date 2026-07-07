# wiShell - 第 4 步

## 已实现内容

- 基于 PTY 的交互式 SSH shell，使用 `RequestPty` 和 `Shell`。
- WebSocket `/ws/terminal` 桥接原始终端 I/O。
- 全双工流式传输：
  - 前端输入 -> SSH stdin
  - SSH stdout/stderr -> 前端输出
- 支持终端 resize：
  - 前端 resize -> WebSocket resize -> SSH `WindowChange`
- 并发模型：
  - 每个连接有 ws 写循环、stdout 读循环、stderr 读循环和 shell wait 循环。
- 处理 UTF-8 多字节边界。

## 协议

1. 客户端发送输入

```json
{
  "type": "input",
  "data": { "text": "ls\r" }
}
```

2. 客户端发送 resize

```json
{
  "type": "resize",
  "data": { "cols": 140, "rows": 40 }
}
```

3. 服务端返回输出

```json
{
  "type": "output",
  "data": {
    "text": "total 8\r\n-rw-r--r-- 1 root root 0 file.txt\r\n",
    "stderr": false
  }
}
```

## 运行

1. 启动后端：

```powershell
go run ./cmd/wiShell
```

2. 启动前端：

```powershell
cd ../frontend
npm run dev
```

3. 打开：

```text
http://localhost:5173
```

4. 连接有效 Linux 服务器后验证：

- `pwd`
- `top`
- `vim`
- 调整窗口大小，确认终端尺寸保持同步。

## 说明

第 4 步已经进入交互式 PTY 模式；终端行为受远程 shell 设置影响。若命令输出延迟，需要检查网络延迟和远程服务器负载。
