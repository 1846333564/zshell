# zShell - 第 2 步

## 已实现内容

- WebSocket 终端端点：`/ws/terminal`。
- 消息协议：`input`、`output`、`error`、`ping`、`pong`、`resize`。
- 前端通过 WebSocket 发送的输入会转换为按行缓冲的命令。
- 命令执行复用 SSH 服务，并通过 WebSocket 返回输出。
- 每个 WebSocket 连接使用独立 goroutine 处理读取、命令执行和写入。

## 端点

- WebSocket URL：`ws://127.0.0.1:8080/ws/terminal?connectionId=<id>`

## 消息协议

1. 客户端发送输入

```json
{
  "type": "input",
  "data": {
    "text": "pwd\n"
  }
}
```

2. 服务端返回输出

```json
{
  "type": "output",
  "data": {
    "text": "/home/root\n",
    "stderr": false
  }
}
```

3. 服务端返回错误

```json
{
  "type": "error",
  "data": {
    "code": "SSH_EXEC_FAILED",
    "message": "dial ssh: ..."
  }
}
```

4. Ping/Pong

- 客户端发送：`{ "type": "ping" }`
- 服务端返回：`{ "type": "pong" }`

5. Resize

该阶段为后续 PTY 保留 resize 消息，服务端返回 `resize-ack`。

## 运行

1. 启动后端：

```powershell
go mod tidy
go run ./cmd/zshell
```

2. 通过第 1 步 API 创建连接。
3. 使用 connectionId 连接 WebSocket。

## PowerShell 冒烟测试

```powershell
$body = @{ name='demo'; host='127.0.0.1'; port=22; username='root'; password='pass' } | ConvertTo-Json
$created = Invoke-RestMethod -Method POST -Uri http://127.0.0.1:8080/api/connections -ContentType 'application/json' -Body $body
$id = $created.connection.id

$ws = [System.Net.WebSockets.ClientWebSocket]::new()
$uri = [Uri]("ws://127.0.0.1:8080/ws/terminal?connectionId=" + $id)
$ws.ConnectAsync($uri, [Threading.CancellationToken]::None).GetAwaiter().GetResult() | Out-Null

$msg = '{"type":"input","data":{"text":"pwd\\n"}}'
```
