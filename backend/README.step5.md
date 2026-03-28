# zShell - Step 5

## What is implemented

### Backend

- SFTP service over SSH
- List remote directory endpoint
- Upload file endpoint
- Download file endpoint

### Frontend

- Basic remote file manager panel
- Directory navigation
- File upload to current directory
- File download from remote entries

## Backend APIs

1. List directory

- POST /api/sftp/list
- Body:

{
  "connectionId": "conn-xxxx",
  "path": "/home/root"
}

- Response:

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

2. Upload file

- POST /api/sftp/upload
- Content-Type: multipart/form-data
- Fields:
  - connectionId
  - path
  - file

3. Download file

- GET /api/sftp/download?connectionId=conn-xxxx&path=/home/root/demo.txt

## Run

1. Start backend

cd backend
go run ./cmd/zshell

2. Start frontend

cd ../frontend
npm run dev

3. Open

http://localhost:5173

## Manual verification checklist

1. Connect to a valid SSH server from the left panel.
2. Confirm terminal still works (pwd, ls, top).
3. In 文件管理 panel, click 刷新 and ensure entries are loaded.
4. Enter a directory and verify list changes.
5. Upload a local file and refresh to confirm presence.
6. Download a remote file and verify local browser download.

## Notes

- The file manager is intentionally basic for MVP and avoids database/storage.
- All operations are local client -> local Go backend -> remote SSH server.
