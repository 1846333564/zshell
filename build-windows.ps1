$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$frontendDir = Join-Path $root 'frontend'
$backendDir = Join-Path $root 'backend'
$frontendDist = Join-Path $frontendDir 'dist'
$embeddedApp = Join-Path $backendDir 'internal\web\app'
$outputExe = 'D:\zshell.exe'

Push-Location $frontendDir
try {
  npm ci
  npm run build
} finally {
  Pop-Location
}

$resolvedEmbeddedApp = [System.IO.Path]::GetFullPath($embeddedApp)
$expectedEmbeddedApp = [System.IO.Path]::GetFullPath((Join-Path $root 'backend\internal\web\app'))
if ($resolvedEmbeddedApp -ne $expectedEmbeddedApp) {
  throw "Refusing to clean unexpected path: $resolvedEmbeddedApp"
}

New-Item -ItemType Directory -Force -Path $embeddedApp | Out-Null
Get-ChildItem -LiteralPath $embeddedApp -Force | Where-Object { $_.Name -ne '.gitkeep' } | Remove-Item -Recurse -Force
Copy-Item -Path (Join-Path $frontendDist '*') -Destination $embeddedApp -Recurse -Force

Push-Location $backendDir
try {
  go test ./...
  $wails = Join-Path $env:USERPROFILE 'go\bin\wails.exe'
  if (!(Test-Path -LiteralPath $wails)) {
    go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
  }
  & $wails build -clean -s -skipbindings -o zshell.exe -webview2 embed -ldflags '-s -w'
  $builtExe = Join-Path $backendDir 'build\bin\zshell.exe'
  if (!(Test-Path -LiteralPath $builtExe)) {
    throw "Wails build did not produce $builtExe"
  }
  Get-Process | Where-Object { $_.Path -eq $outputExe } | Stop-Process -Force
  Copy-Item -LiteralPath $builtExe -Destination $outputExe -Force
} finally {
  Pop-Location
}

Write-Host "Built $outputExe"
