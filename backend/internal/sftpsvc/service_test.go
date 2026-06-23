package sftpsvc

import "testing"

func TestNormalizeRemotePath(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: "~"},
		{name: "spaces", input: "   ", want: "~"},
		{name: "root", input: "/", want: "/"},
		{name: "normal", input: "/home/root", want: "/home/root"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeRemotePath(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeRemotePath(%q)=%q, want=%q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCleanRelativePath(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		fallback string
		want     string
		wantErr  bool
	}{
		{name: "fallback", input: "", fallback: "demo.txt", want: "demo.txt"},
		{name: "nested folder upload", input: "folder/sub/demo.txt", want: "folder/sub/demo.txt"},
		{name: "windows separators", input: `folder\sub\demo.txt`, want: "folder/sub/demo.txt"},
		{name: "leading slash", input: "/folder/demo.txt", want: "folder/demo.txt"},
		{name: "path traversal", input: "../demo.txt", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := cleanRelativePath(tc.input, tc.fallback)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("cleanRelativePath(%q)=%q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCleanZipPathDropsUnsafeSegments(t *testing.T) {
	got := cleanZipPath(`/safe/../folder\demo.txt`)
	if got != "safe/folder/demo.txt" {
		t.Fatalf("unexpected zip path: %q", got)
	}
}
