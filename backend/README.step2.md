# zShell - Step 2

## What is implemented

- WebSocket terminal endpoint: /ws/terminal
- Message protocol implemented: input, output, error, ping, pong, resize
- Frontend input over WebSocket is converted to command lines (line-buffered)
- Command execution uses existing SSH service and returns output over WebSocket
- Per WebSocket connection uses independent goroutines for:
  - ws read loop
  - command execution loop
  - ws write loop

## Endpoint

- WebSocket URL: ws://127.0.0.1:8080/ws/terminal?connectionId=<id>

## Message protocol

1. Client -> Server input

{
  "type": "input",
  "data": {
    "text": "pwd\n"
  }
}

2. Server -> Client output

{
  "type": "output",
  "data": {
    "text": "/home/root\n",
    "stderr": false
  }
}

3. Server -> Client error

{
  "type": "error",
  "data": {
    "code": "SSH_EXEC_FAILED",
    "message": "dial ssh: ..."
  }
}

4. Ping/Pong

- Client sends: { "type": "ping" }
- Server replies: { "type": "pong" }

5. Resize (reserved for Step 4 PTY)

- Client sends resize payload.
- Server responds with type "resize-ack".

## Run

1. Start backend

   go mod tidy
   go run ./cmd/zshell

2. Create connection profile via HTTP (Step 1 API)

3. Connect to WebSocket URL with connectionId

## PowerShell smoke test

$body = @{ name='demo'; host='127.0.0.1'; port=22; username='root'; password='pass' } | ConvertTo-Json
$created = Invoke-RestMethod -Method POST -Uri http://127.0.0.1:8080/api/connections -ContentType 'application/json' -Body $body
$id = $created.connection.id

$ws = [System.Net.WebSockets.ClientWebSocket]::new()
$uri = [Uri]("ws://127.0.0.1:8080/ws/terminal?connectionId=" + $id)
$ws.ConnectAsync($uri, [Threading.CancellationToken]::None).GetAwaiter().GetResult() | Out-Null

$msg = '{"type":"input","data":{"text":"pwd\\n"}}'
$bytes = [Text.Encoding]::UTF8.GetBytes($msg)
$ws.SendAsync([ArraySegment[byte]]::new($bytes), [System.Net.WebSockets.WebSocketMessageType]::Text, $true, [Threading.CancellationToken]::None).GetAwaiter().GetResult() | Out-Null

$buffer = New-Object byte[] 8192
$res = $ws.ReceiveAsync([ArraySegment[byte]]::new($buffer), [Threading.CancellationToken]::None).GetAwaiter().GetResult()
[Text.Encoding]::UTF8.GetString($buffer, 0, $res.Count)
$ws.Dispose()

## Note

This step focuses on WebSocket transport. Full interactive PTY behavior for top/vim is planned in Step 4.
