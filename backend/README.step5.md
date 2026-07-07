# wiShell - 第 5 步

## 已实现内容

### 后端

- 基于 SSH 的 SFTP 服务。
- 远程目录列表接口。
- 文件上传接口。
- 文件下载接口。

### 前端

- 基础远程文件管理器面板。
- 目录导航。
- 上传文件到当前目录。
- 从远程条目下载文件。

## 后端 API

1. 列出目录

- `POST /api/sftp/list`
- 请求体：

```json
{
  "connectionId": "conn-xxxx",
  "path": "/home/root"
}
```

- 响应：

```json
{
  "path": "/home/root",
  "entries": [
    {
      "name": "demo.txt",
      "path": "/home/root/demo.txt",
      "size": 12,
      "isDir": false,
      "mode": "-rw-r--r--",
      "modTime": "2026-03-28T07:00:00Z"
    }
  ]
}
```

2. 上传文件

- `POST /api/sftp/upload`
- `Content-Type: multipart/form-data`
- 字段：
  - `connectionId`
  - `path`
  - `file`

3. 下载文件

- `GET /api/sftp/download?connectionId=conn-xxxx&path=/home/root/demo.txt`

## 运行

1. 启动后端：

```powershell
cd backend
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

## 手工验证清单

1. 从左侧面板连接有效 SSH 服务器。
2. 确认终端仍正常工作，例如 `pwd`、`ls`、`top`。
3. 在“文件管理”面板点击刷新，确认能加载远程目录条目。
