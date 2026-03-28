# zShell - Step 1

## What is implemented

- Go backend project initialized
- In-memory multi-server connection storage
- SSH connection test (username/password)
- Execute one command over SSH and return stdout/stderr/exitCode


## Start backend

1. Open terminal in backend directory.
2. Run:

   go mod tidy
   go run ./cmd/zshell

Backend listens on http://127.0.0.1:8080.

## API list

- GET /api/health
- POST /api/connections
- GET /api/connections
- POST /api/ssh/test
- POST /api/ssh/exec

## Quick test commands (PowerShell)

1. Health

Invoke-RestMethod -Method GET -Uri http://127.0.0.1:8080/api/health | ConvertTo-Json -Depth 5

2. Add connection

$body = @{ name='demo'; host='192.168.1.10'; port=22; username='root'; password='your-password' } | ConvertTo-Json
Invoke-RestMethod -Method POST -Uri http://127.0.0.1:8080/api/connections -ContentType 'application/json' -Body $body | ConvertTo-Json -Depth 5

3. List connections

Invoke-RestMethod -Method GET -Uri http://127.0.0.1:8080/api/connections | ConvertTo-Json -Depth 5

4. Test SSH connection

$body = @{ connectionId='replace-with-connection-id' } | ConvertTo-Json
Invoke-RestMethod -Method POST -Uri http://127.0.0.1:8080/api/ssh/test -ContentType 'application/json' -Body $body | ConvertTo-Json -Depth 5

5. Execute one command

$body = @{ connectionId='replace-with-connection-id'; command='pwd' } | ConvertTo-Json
Invoke-RestMethod -Method POST -Uri http://127.0.0.1:8080/api/ssh/exec -ContentType 'application/json' -Body $body | ConvertTo-Json -Depth 6
