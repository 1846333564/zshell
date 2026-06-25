package logsvc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoggerCleansOldLogsAndRotatesHourly(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 6, 25, 14, 30, 0, 0, time.Local)

	oldPath := filepath.Join(dir, "zshell-20260624-13.log")
	keepPath := filepath.Join(dir, "zshell-20260624-15.log")
	if err := os.WriteFile(oldPath, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keepPath, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldTime := now.Add(-25 * time.Hour)
	keepTime := now.Add(-23 * time.Hour)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(keepPath, keepTime, keepTime); err != nil {
		t.Fatal(err)
	}

	logger, err := newWithClock(dir, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old log was not removed: %v", err)
	}
	if _, err := os.Stat(keepPath); err != nil {
		t.Fatalf("recent log should remain: %v", err)
	}

	if _, err := logger.Write([]byte("first\n")); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Hour)
	if _, err := logger.Write([]byte("second\n")); err != nil {
		t.Fatal(err)
	}

	first, err := os.ReadFile(filepath.Join(dir, "zshell-20260625-14.log"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(filepath.Join(dir, "zshell-20260625-15.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(first), "first") || !strings.Contains(string(second), "second") {
		t.Fatalf("hourly logs did not contain expected content")
	}
}
