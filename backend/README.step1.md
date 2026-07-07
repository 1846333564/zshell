# wiShell - 第 1 步

## 已实现内容

- 初始化 Go 后端项目。
- 实现内存中的多服务器连接存储。
- 实现 SSH 连接测试，支持用户名和密码。
- 支持通过 SSH 执行单条命令，并返回 stdout、stderr 和 exitCode。

## 启动后端

1. 在 `backend` 目录打开终端。
2. 运行：

```powershell
go mod tidy
go run ./cmd/wiShell
```

后端监听地址为 `http://127.0.0.1:8080`。

## API 列表

- `GET /api/health`
- `POST /api/connections`
- `GET /api/connections`
- `POST /api/ssh/test`
- `POST /api/ssh/exec`

## PowerShell 快速测试

1. 健康检查

```powershell
Invoke-RestMethod -Method GET -Uri http://127.0.0.1:8080/api/health | ConvertTo-Json -Depth 5
```

2. 新增连接

```powershell
$body = @{ name='demo'; host='192.168.1.10'; port=22; username='root'; password='your-password' } | ConvertTo-Json
Invoke-RestMethod -Method POST -Uri http://127.0.0.1:8080/api/connections -ContentType 'application/json' -Body $body | ConvertTo-Json -Depth 5
```

3. 查看连接

```powershell
Invoke-RestMethod -Method GET -Uri http://127.0.0.1:8080/api/connections | ConvertTo-Json -Depth 5
```

4. 测试 SSH 连接

```powershell
$body = @{ connectionId='replace-with-connection-id' } | ConvertTo-Json
Invoke-RestMethod -Method POST -Uri http://127.0.0.1:8080/api/ssh/test -ContentType 'application/json' -Body $body | ConvertTo-Json -Depth 5
```

5. 执行单条命令

```powershell
$body = @{ connectionId='replace-with-connection-id'; command='pwd' } | ConvertTo-Json
Invoke-RestMethod -Method POST -Uri http://127.0.0.1:8080/api/ssh/exec -ContentType 'application/json' -Body $body | ConvertTo-Json -Depth 6
```
