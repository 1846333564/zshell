# 应用信息模块状态

## 范围

集中保存 zShell 产品信息、当前版本号和 GitHub Release 仓库配置。

## 重要文件

- `info.go`

## 当前状态

`appinfo.Version` 默认是 `0.0.1`，release 构建时由 `build-windows.ps1` 读取根目录 `VERSION` 并通过 Go ldflags 注入。模块同时提供产品名、公司、开发者、内测状态、GitHub owner/repo 和 release exe 命名规则。

## 已知工作

如果未来支持多发布通道，需要把 channel、仓库和资产规则扩展为可配置项。
