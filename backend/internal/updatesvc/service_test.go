package updatesvc

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name  string
		left  string
		right string
		want  int
	}{
		{name: "patch newer", left: "0.0.2", right: "0.0.1", want: 1},
		{name: "same with v prefix", left: "v0.0.1", right: "0.0.1", want: 0},
		{name: "minor newer", left: "0.1.0", right: "0.0.9", want: 1},
		{name: "major older", left: "0.9.9", right: "1.0.0", want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareVersions(tt.left, tt.right)
			if got != tt.want {
				t.Fatalf("compareVersions(%q, %q) = %d, want %d", tt.left, tt.right, got, tt.want)
			}
		})
	}
}

func TestFindExecutableAsset(t *testing.T) {
	assets := []githubAsset{
		{Name: "wiShell.0.0.1.exe.sha256"},
		{Name: "wiShell.0.0.1.exe", BrowserDownloadURL: "https://example.test/wiShell.0.0.1.exe"},
	}

	asset := findExecutableAsset(assets, "0.0.1")
	if asset.Name != "wiShell.0.0.1.exe" {
		t.Fatalf("asset.Name = %q, want wiShell.0.0.1.exe", asset.Name)
	}
}

func TestExplainDownloadErrorIncludesManualURL(t *testing.T) {
	err := explainDownloadError("下载更新失败", "https://example.test/wiShell.exe", assertErr("timeout"))
	if err == nil {
		t.Fatal("explainDownloadError returned nil")
	}
	got := err.Error()
	if !containsAll(got, []string{"下载更新失败", "https://example.test/wiShell.exe", "timeout"}) {
		t.Fatalf("error message %q does not include expected context", got)
	}
}

func TestStopIfCanceledReturnsStoppedError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := stopIfCanceled(ctx)
	if !errors.Is(err, ErrStopped) {
		t.Fatalf("stopIfCanceled() = %v, want ErrStopped", err)
	}
}

func TestExplainDownloadErrorKeepsStoppedError(t *testing.T) {
	err := explainDownloadError("下载更新失败", "https://example.test/wiShell.exe", ErrStopped)
	if !errors.Is(err, ErrStopped) {
		t.Fatalf("explainDownloadError() = %v, want ErrStopped", err)
	}
}

func TestManualReleaseURL(t *testing.T) {
	want := "https://github.com/1846333564/zshell/releases/latest"
	if got := manualReleaseURL(); got != want {
		t.Fatalf("manualReleaseURL() = %q, want %q", got, want)
	}
}

func TestDownloadPercent(t *testing.T) {
	tests := []struct {
		name   string
		loaded int64
		total  int64
		want   int
	}{
		{name: "start with total", loaded: 0, total: 100, want: 32},
		{name: "half downloaded", loaded: 50, total: 100, want: 57},
		{name: "complete", loaded: 100, total: 100, want: 82},
		{name: "unknown total without bytes", loaded: 0, total: 0, want: 32},
		{name: "unknown total with bytes", loaded: 2048, total: 0, want: 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := downloadPercent(tt.loaded, tt.total); got != tt.want {
				t.Fatalf("downloadPercent(%d, %d) = %d, want %d", tt.loaded, tt.total, got, tt.want)
			}
		})
	}
}

type assertErr string

func (e assertErr) Error() string {
	return string(e)
}

func containsAll(value string, parts []string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
