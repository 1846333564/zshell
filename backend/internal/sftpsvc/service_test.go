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

func TestCopyPathCandidate(t *testing.T) {
	cases := []struct {
		name  string
		input string
		index int
		want  string
	}{
		{name: "file first copy", input: "/home/demo.txt", index: 1, want: "/home/demo copy.txt"},
		{name: "file second copy", input: "/home/demo.txt", index: 2, want: "/home/demo copy 2.txt"},
		{name: "directory copy", input: "/home/demo", index: 1, want: "/home/demo copy"},
		{name: "dotfile copy", input: "/home/.env", index: 1, want: "/home/.env copy"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := copyPathCandidate(tc.input, tc.index)
			if got != tc.want {
				t.Fatalf("copyPathCandidate(%q, %d)=%q, want %q", tc.input, tc.index, got, tc.want)
			}
		})
	}
}

func TestIsSameOrChildPath(t *testing.T) {
	cases := []struct {
		name      string
		candidate string
		parent    string
		want      bool
	}{
		{name: "same path", candidate: "/home/demo", parent: "/home/demo", want: true},
		{name: "child path", candidate: "/home/demo/sub", parent: "/home/demo", want: true},
		{name: "sibling prefix is not child", candidate: "/home/demo2", parent: "/home/demo", want: false},
		{name: "parent path", candidate: "/home", parent: "/home/demo", want: false},
		{name: "root contains absolute path", candidate: "/home/demo", parent: "/", want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isSameOrChildPath(tc.candidate, tc.parent)
			if got != tc.want {
				t.Fatalf("isSameOrChildPath(%q, %q)=%t, want %t", tc.candidate, tc.parent, got, tc.want)
			}
		})
	}
}

func TestIsProtectedDeletePath(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "empty", input: "", want: true},
		{name: "spaces", input: "   ", want: true},
		{name: "dot", input: ".", want: true},
		{name: "root", input: "/", want: true},
		{name: "normal file", input: "/home/demo.txt", want: false},
		{name: "normal directory", input: "/home/demo", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isProtectedDeletePath(tc.input)
			if got != tc.want {
				t.Fatalf("isProtectedDeletePath(%q)=%t, want %t", tc.input, got, tc.want)
			}
		})
	}
}

func TestDeleteShellPathArg(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		wantRemote string
		wantArg    string
		wantErr    bool
	}{
		{name: "absolute path", input: "/home/demo file.txt", wantRemote: "/home/demo file.txt", wantArg: "'/home/demo file.txt'"},
		{name: "home child path", input: "~/demo file.txt", wantRemote: "~/demo file.txt", wantArg: "${HOME}'/demo file.txt'"},
		{name: "quote escaping", input: "/home/a'b.txt", wantRemote: "/home/a'b.txt", wantArg: "'/home/a'\"'\"'b.txt'"},
		{name: "root rejected", input: "/", wantErr: true},
		{name: "home rejected", input: "~", wantErr: true},
		{name: "relative rejected", input: "demo.txt", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotRemote, gotArg, err := deleteShellPathArg(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotRemote != tc.wantRemote || gotArg != tc.wantArg {
				t.Fatalf("deleteShellPathArg(%q)=(%q,%q), want (%q,%q)", tc.input, gotRemote, gotArg, tc.wantRemote, tc.wantArg)
			}
		})
	}
}

func TestSameConnectionTransferCommand(t *testing.T) {
	copyCommand := sameConnectionTransferCommand("copy", "/home/source file.txt", "/home/target file.txt")
	wantCopy := "cp -a --reflink=auto -- '/home/source file.txt' '/home/target file.txt' 2>/dev/null || cp -a -- '/home/source file.txt' '/home/target file.txt'"
	if copyCommand != wantCopy {
		t.Fatalf("copy command=%q, want %q", copyCommand, wantCopy)
	}

	moveCommand := sameConnectionTransferCommand("move", "/home/source file.txt", "/home/target file.txt")
	wantMove := "mv -f -- '/home/source file.txt' '/home/target file.txt'"
	if moveCommand != wantMove {
		t.Fatalf("move command=%q, want %q", moveCommand, wantMove)
	}
}
