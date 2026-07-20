# HTTP API 模块状态

## 范围

本地 HTTP 路由，覆盖健康检查、应用信息、更新检查与应用、连接生命周期、SSH、SFTP、远程文本编辑、跨服务器传输、归档下载和监控快照。

## 重要文件

- `server.go`：服务结构和路由注册。
- `requests.go`：请求 DTO。
- `handlers_*.go`：按系统、更新、连接、SSH、SFTP 和监控拆分的 HTTP handler。
- `config.go`、`multipart.go`、`response.go`：配置归一化、上传解析和响应工具。

## 当前状态

路由注册在标准 `http.ServeMux` 上。连接配置路由位于 `/api/config/connections`，支持 `GET`、`POST`、`PUT` 和 `DELETE`，连接字段包含认证方式、工作模式 `workMode` 和最近一次读取的服务器硬件信息；UI 偏好路由位于 `/api/config/preferences`，支持 `GET` 和 `PUT`，当前保存 `uiScale`、`terminalFontSize`、`themeKey`、`customTheme` 和 `gpuAccelerationEnabled`，其中主题键会归一化到内置主题或 `custom`，自定义颜色只保存合法 `#RRGGBB` 值，GPU 设置缺失时默认开启。保存配置由 `configstore` 持久化，活动连接查询使用 `store.MemoryStore`。密码连接编辑时，如果密码为空，会保留旧密码。

`/api/ssh/test` 在 SSH 连通后会读取服务器硬件信息并写回连接配置。SFTP 上传支持多文件和多目录；`/api/sftp/upload/stream` 会以 NDJSON 返回后端真实 SFTP 写入进度，本地 API 入口不再用 30 秒写超时限制这类长时间流式请求。远程文本编辑使用 `/api/sftp/file/read`、`/api/sftp/file/read/raw`、兼容保留的 `/api/sftp/file/read/stream` 和 `/api/sftp/file/write`；当前编辑器读取走 raw 接口，以原始 UTF-8 响应、独立的预期文件大小以及编码后的路径/名称/修改时间响应头返回最多 256 MiB 内容。raw handler 只读取打开时记录的快照长度，显式刷新响应头和每个正文块，并监听请求上下文；浏览器取消或关闭连接时会关闭远程 SFTP 文件和会话，不再继续下载，CORS 会显式暴露元数据响应头。远程文件传输使用 `/api/sftp/transfer`，同服务器复制/移动由后端走远端快路径，跨服务器保留 SFTP 流式传输，同名移动目标现在返回 HTTP 409 而不是覆盖或合并；`POST /api/sftp/rename` 接受 `connectionId`、`path` 和 `newName`，安全重命名同父目录项目，非法名称或受保护路径返回 400，目标已存在返回 409，并返回旧路径、新路径、名称、目录类型和是否实际变更；远程文件删除使用 `/api/sftp/delete` 强制递归删除选中项，同服务器删除走远端快路径并继续拒绝危险路径；监控快照由 `POST /api/monitor/snapshot` 返回。应用信息由 `GET /api/app/info` 返回，更新检查和应用分别使用 `POST /api/update/check` 与 `POST /api/update/apply`，前端确认更新使用 `POST /api/update/apply/stream` 接收 NDJSON 进度事件；如果更新请求被停止，stream handler 返回 stopped 事件且不按错误日志记录。

所有 `writeError` 返回的 API 错误会记录到日志系统，日志包含 handler 调用位置和原始错误消息；HTTP middleware 会捕获并记录 panic。流式上传、流式远程文本读取、流式更新和归档下载在响应已开始后失败时会额外显式记录日志。

## 已知工作

配置增删改查仍需要更细的 handler 测试，尤其是密码保留和配置存储失败场景。更新接口需要在真实 GitHub Release 存在后做端到端验证。
