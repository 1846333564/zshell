# SSH 服务模块状态

## 范围

SSH 客户端创建、认证方式选择、一次性命令执行和交互式 PTY shell。

## 重要文件

- `client.go`
- `shell.go`

## 当前状态

支持密码认证和当前 Windows 用户的 `~/.ssh/id_rsa` 私钥认证。PTY shell 为终端 WebSocket 提供交互式能力，并定期发送 SSH 全局 keepalive 请求，降低长时间交互终端断线概率。

## 已知工作

Host key 校验目前仍是宽松模式。私钥模式有意限制为 `~/.ssh/id_rsa`。如果远程 shell 自己通过 `TMOUT` 等策略强制空闲登出，传输层 keepalive 不会覆盖服务器策略。
