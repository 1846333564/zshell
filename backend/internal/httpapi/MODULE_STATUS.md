# HTTP API 模块状态

## 范围

本地 HTTP 路由，覆盖健康检查、应用信息、更新检查与应用、连接生命周期、SSH、SFTP、远程文本编辑、跨服务器传输、归档下载和监控快照。

## 重要文件

- `server.go`

## 当前状态

路由注册在标准 `http.ServeMux` 上。连接配置路由位于 `/api/config/connections`，支持 `GET`、`POST`、`PUT` 和 `DELETE`；UI 偏好路由位于 `/api/config/preferences`，支持 `GET` 和 `PUT`。保存配置由 `configstore` 持久化，活动连接查询使用 `store.MemoryStore`。密码连接编辑时，如果密码为空，会保留旧密码。

SFTP 上传支持多文件和多目录；远程文本编辑使用 `/api/sftp/file/read` 和 `/api/sftp/file/write`；监控快照由 `POST /api/monitor/snapshot` 返回。应用信息由 `GET /api/app/info` 返回，更新检查和应用分别使用 `POST /api/update/check` 与 `POST /api/update/apply`。

## 已知工作

配置增删改查仍需要更细的 handler 测试，尤其是密码保留和配置存储失败场景。更新接口需要在真实 GitHub Release 存在后做端到端验证。
