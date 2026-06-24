package updatesvc

import "testing"

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
		{Name: "zshell.0.0.1.exe.sha256"},
		{Name: "zshell.0.0.1.exe", BrowserDownloadURL: "https://example.test/zshell.0.0.1.exe"},
	}

	asset := findExecutableAsset(assets, "0.0.1")
	if asset.Name != "zshell.0.0.1.exe" {
		t.Fatalf("asset.Name = %q, want zshell.0.0.1.exe", asset.Name)
	}
}
