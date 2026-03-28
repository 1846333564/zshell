# zShell - Step 3

## What is implemented

- Vue 3 + Vite frontend scaffold
- xterm.js terminal component integrated
- Connection form implemented
- HTTP API integration for connection create + test
- WebSocket integration with /ws/terminal
- Terminal input/output path wired end-to-end

## Start order

1. Start backend in one terminal

cd backend
go run ./cmd/zshell

2. Start frontend in another terminal

cd frontend
npm install
npm run dev

3. Open browser

http://127.0.0.1:5173

## Usage

1. Fill host, port, username, password.
2. Click 测试并连接.
3. After connected, type command in terminal and press Enter.
4. Output is streamed back to xterm panel.

## Notes

- Step 3 focuses on xterm + websocket I/O.
- Interactive PTY behavior for top/vim will be completed in Step 4.
