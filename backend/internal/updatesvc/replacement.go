package updatesvc

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func scheduleReplacement(sourcePath string) error {
	targetPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate current executable: %w", err)
	}

	scriptPath := filepath.Join(os.TempDir(), fmt.Sprintf("wiShell-apply-update-%d.ps1", time.Now().UnixNano()))
	script := fmt.Sprintf(`$ErrorActionPreference = 'Stop'
$source = '%s'
$target = '%s'
$pidToWait = %d
try {
  Wait-Process -Id $pidToWait -Timeout 60 -ErrorAction SilentlyContinue
} catch {}
Start-Sleep -Milliseconds 800
Copy-Item -LiteralPath $source -Destination $target -Force
Start-Process -FilePath $target
Remove-Item -LiteralPath $source -Force -ErrorAction SilentlyContinue
Remove-Item -LiteralPath $MyInvocation.MyCommand.Path -Force -ErrorAction SilentlyContinue
`, powerShellQuote(sourcePath), powerShellQuote(targetPath), os.Getpid())

	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		return fmt.Errorf("write update helper: %w", err)
	}

	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start update helper: %w", err)
	}
	return nil
}

func powerShellQuote(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
