# 配置存储模块状态

## 范围

当前 Windows 用户下加密保存连接配置和 UI 偏好。

## 重要文件

- `store.go`
- `dpapi_windows.go`
- `dpapi_other.go`

## 当前状态

保存连接和 UI 偏好会通过 `os.UserConfigDir()` 写入 `%AppData%\zShell\connections.dpapi`，并用 Windows DPAPI 针对当前用户加密/解密。日志系统使用同一配置根目录下的 `%AppData%\zShell\log` 独立文件夹。连接配置包含认证方式、工作模式和最近一次服务器硬件信息；当前 UI 偏好包含非终端 UI 缩放、终端字体大小、主题键和自定义主题颜色。非 Windows 构建会返回明确的不支持错误。

## 已知工作

如果该模块继续扩大，需要围绕加载/保存错误处理增加单元测试，并考虑注入路径或加密包装以便测试。
