# 更新服务模块状态

## 范围

通过 GitHub Release 检查、下载并应用 zShell Windows exe 更新。

## 重要文件

- `service.go`

## 当前状态

更新服务请求 `1846333564/zshell` 的最新 GitHub Release，按 `zshell.<版本号>.exe` 查找 Windows 可执行文件资产，比较当前版本和最新版本。如果未认证 GitHub API 触发配额限制，会回退到 GitHub Releases `latest` 页面重定向解析 tag，并用确定性的 Release 下载地址继续检查。应用更新时会下载 exe 到临时目录，若 Release 提供 GitHub digest 或 `.sha256` 校验资产则验证 SHA256，然后启动独立 PowerShell helper，在当前进程退出后替换当前 exe 并重启应用。

## 已知工作

当前更新链路默认 GitHub Release 可公开访问。若仓库或 Release 资产设为私有，需要增加认证和令牌管理。真实替换流程需要在发布新版本后做端到端验证。
