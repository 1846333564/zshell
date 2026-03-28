# zShell - Step 4

## What is implemented

- Interactive SSH shell over PTY (RequestPty + Shell)
- WebSocket `/ws/terminal` now bridges raw terminal I/O
- Full duplex streaming:
  - Frontend input -> SSH stdin
  - SSH stdout/stderr -> Frontend output
- Terminal resize support:
  - Frontend resize -> WebSocket resize -> SSH WindowChange
- Concurrent model:
  - Per connection goroutines for ws write loop, stdout read loop, stderr read loop, and shell wait loop
- UTF-8 stream handling with multi-byte boundary protection

## Protocol

1. Client send input

{
  "type": "input",
  "data": { "text": "ls\r" }
}

2. Client send resize

{
  "type": "resize",
  "data": { "cols": 140, "rows": 40 }
}

3. Server output

{
  "type": "output",
  "data": {
    "text": "total 8\r\n-rw-r--r-- 1 root root 0 file.txt\r\n",
    "stderr": false
  }
}

## Run

1. Start backend

   go run ./cmd/zshell

2. Start frontend

   cd ../frontend
   npm run dev

3. Open browser

   http://localhost:5173

4. Connect to a valid Linux server and verify:

- `pwd`
- `top`
- `vim`
- Resize browser window and check terminal stays synced

## Notes

- Step 4 is now interactive PTY mode; terminal behavior depends on remote shell settings.
- If commands appear delayed, check network latency and remote server load.
