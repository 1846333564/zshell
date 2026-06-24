$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$frontendDir = Join-Path $root 'frontend'
$backendDir = Join-Path $root 'backend'
$frontendDist = Join-Path $frontendDir 'dist'
$embeddedApp = Join-Path $backendDir 'internal\web\app'
$versionPath = Join-Path $root 'VERSION'
$version = (Get-Content -LiteralPath $versionPath -Raw).Trim()
$releaseDir = Join-Path $root 'release'
$outputExe = Join-Path $releaseDir "zshell.$version.exe"

if ($version -notmatch '^\d+\.\d+\.\d+$') {
  throw "Invalid VERSION value: $version"
}

function Invoke-Native {
  param(
    [Parameter(Mandatory = $true)]
    [string] $Description,
    [Parameter(Mandatory = $true)]
    [scriptblock] $Command
  )

  $global:LASTEXITCODE = 0
  & $Command
  if ($LASTEXITCODE -ne 0) {
    throw "$Description failed with exit code $LASTEXITCODE"
  }
}

Push-Location $frontendDir
try {
  Invoke-Native 'npm ci' { npm ci }
  Invoke-Native 'npm run build' { npm run build }
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
  Invoke-Native 'go test ./...' { go test ./... }
  $wails = Join-Path $env:USERPROFILE 'go\bin\wails.exe'
  if (!(Test-Path -LiteralPath $wails)) {
    Invoke-Native 'go install Wails CLI' { go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 }
  }
  $ldflags = "-s -w -X zshell/backend/internal/appinfo.Version=$version"
  Invoke-Native 'wails build' { & $wails build -clean -s -skipbindings -o zshell.exe -webview2 embed -ldflags $ldflags }
  $builtExe = Join-Path $backendDir 'build\bin\zshell.exe'
  if (!(Test-Path -LiteralPath $builtExe)) {
    throw "Wails build did not produce $builtExe"
  }
  New-Item -ItemType Directory -Force -Path $releaseDir | Out-Null
  $releaseRoot = [System.IO.Path]::GetFullPath($releaseDir).TrimEnd('\') + '\'
  $runningOutput = Get-Process | Where-Object {
    $_.Path -and [System.IO.Path]::GetFullPath($_.Path).StartsWith($releaseRoot, [System.StringComparison]::OrdinalIgnoreCase)
  }
  if ($runningOutput) {
    $runningOutput | Stop-Process -Force
    foreach ($process in $runningOutput) {
      Wait-Process -Id $process.Id -Timeout 5 -ErrorAction SilentlyContinue
    }
    Start-Sleep -Milliseconds 300
  }
  Get-ChildItem -LiteralPath $releaseDir -Filter '*.exe' -File -ErrorAction SilentlyContinue | Remove-Item -Force
  Copy-Item -LiteralPath $builtExe -Destination $outputExe -Force
} finally {
  Pop-Location
}

Write-Host "Built $outputExe"
