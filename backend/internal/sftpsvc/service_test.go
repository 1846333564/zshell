package sftpsvc

import (
	"bytes"
	"errors"
	"slices"
	"testing"
)

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

func TestPrepareUploadBatchDeduplicatesDirectories(t *testing.T) {
	files := []UploadItem{
		{FileName: "index.html", RelativePath: "app/index.html", Size: 10},
		{FileName: "main.js", RelativePath: "app/assets/main.js", Size: 20},
		{FileName: "logo.svg", RelativePath: `app\assets\logo.svg`, Size: 30},
	}
	directories := []string{"app", "app/assets", "app/assets"}

	prepared, explicitDirs, dirsToCreate, err := prepareUploadBatch("/srv/www", files, directories)
	if err != nil {
		t.Fatalf("prepareUploadBatch returned error: %v", err)
	}

	wantPrepared := []string{
		"/srv/www/app/index.html",
		"/srv/www/app/assets/main.js",
		"/srv/www/app/assets/logo.svg",
	}
	gotPrepared := make([]string, 0, len(prepared))
	for _, item := range prepared {
		gotPrepared = append(gotPrepared, item.remotePath)
	}
	if !slices.Equal(gotPrepared, wantPrepared) {
		t.Fatalf("prepared remote paths=%v, want %v", gotPrepared, wantPrepared)
	}

	wantExplicitDirs := []string{"/srv/www/app", "/srv/www/app/assets", "/srv/www/app/assets"}
	if !slices.Equal(explicitDirs, wantExplicitDirs) {
		t.Fatalf("explicit dirs=%v, want %v", explicitDirs, wantExplicitDirs)
	}

	wantDirsToCreate := []string{"/srv/www/app", "/srv/www/app/assets"}
	if !slices.Equal(dirsToCreate, wantDirsToCreate) {
		t.Fatalf("dirs to create=%v, want %v", dirsToCreate, wantDirsToCreate)
	}
}

func TestUploadWorkerCountIsBounded(t *testing.T) {
	if got := uploadWorkerCount(3); got != 3 {
		t.Fatalf("uploadWorkerCount(3)=%d, want 3", got)
	}
	if got := uploadWorkerCount(1000); got != uploadFileWorkerLimit {
		t.Fatalf("uploadWorkerCount(1000)=%d, want %d", got, uploadFileWorkerLimit)
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
	wantMove := "mv -nT -- '/home/source file.txt' '/home/target file.txt' && if [ -e '/home/source file.txt' ] || [ -L '/home/source file.txt' ]; then echo 'move refused because target exists' >&2; exit 1; fi"
	if moveCommand != wantMove {
		t.Fatalf("move command=%q, want %q", moveCommand, wantMove)
	}
}

func TestValidateRenameName(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "plain name", input: "renamed.txt"},
		{name: "spaces in name", input: "renamed file.txt"},
		{name: "dotfile", input: ".env"},
		{name: "backslash is a valid POSIX name character", input: `name\part`},
		{name: "empty", input: "", wantErr: true},
		{name: "spaces only", input: "   ", wantErr: true},
		{name: "dot", input: ".", wantErr: true},
		{name: "dot dot", input: "..", wantErr: true},
		{name: "slash", input: "nested/name", wantErr: true},
		{name: "nul", input: "bad\x00name", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRenameName(tc.input)
			if tc.wantErr {
				if !errors.Is(err, ErrInvalidRenameName) {
					t.Fatalf("validateRenameName(%q) error=%v, want ErrInvalidRenameName", tc.input, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateRenameName(%q) unexpected error: %v", tc.input, err)
			}
		})
	}
}

func TestValidateRenameSourcePath(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "absolute item", input: "/opt/demo.txt"},
		{name: "home child", input: "~/demo.txt"},
		{name: "empty", input: "", wantErr: ErrInvalidRenamePath},
		{name: "root", input: "/", wantErr: ErrProtectedRenamePath},
		{name: "cleaned root", input: "/opt/..", wantErr: ErrProtectedRenamePath},
		{name: "home", input: "~", wantErr: ErrProtectedRenamePath},
		{name: "relative", input: "demo.txt", wantErr: ErrInvalidRenamePath},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRenameSourcePath(tc.input)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("validateRenameSourcePath(%q) error=%v, want %v", tc.input, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateRenameSourcePath(%q) unexpected error: %v", tc.input, err)
			}
		})
	}
}

func TestRenameTargetPathStaysInSourceParent(t *testing.T) {
	got, err := renameTargetPath("/opt/app/old.txt", "new.txt")
	if err != nil {
		t.Fatalf("renameTargetPath returned error: %v", err)
	}
	if got != "/opt/app/new.txt" {
		t.Fatalf("renameTargetPath=%q, want /opt/app/new.txt", got)
	}
	got, err = renameTargetPath("/opt/app/old ", "new ")
	if err != nil {
		t.Fatalf("renameTargetPath with trailing spaces returned error: %v", err)
	}
	if got != "/opt/app/new " {
		t.Fatalf("renameTargetPath with trailing spaces=%q, want %q", got, "/opt/app/new ")
	}

	if _, err := renameTargetPath("/", "new-root"); !errors.Is(err, ErrProtectedRenamePath) {
		t.Fatalf("root rename error=%v, want ErrProtectedRenamePath", err)
	}
	if _, err := renameTargetPath("/opt/app/old.txt", "nested/new.txt"); !errors.Is(err, ErrInvalidRenameName) {
		t.Fatalf("nested rename error=%v, want ErrInvalidRenameName", err)
	}
}

func TestEnsureMoveTargetAvailable(t *testing.T) {
	t.Run("same source and target is no-op", func(t *testing.T) {
		called := false
		noOp, err := ensureMoveTargetAvailable("/opt/demo", "/opt/demo", true, func(string) (bool, error) {
			called = true
			return true, nil
		})
		if err != nil || !noOp || called {
			t.Fatalf("noOp=%t called=%t err=%v, want true false nil", noOp, called, err)
		}
	})

	t.Run("existing target is conflict", func(t *testing.T) {
		noOp, err := ensureMoveTargetAvailable("/opt/source", "/srv/source", true, func(string) (bool, error) {
			return true, nil
		})
		if noOp || !errors.Is(err, ErrTargetExists) {
			t.Fatalf("noOp=%t err=%v, want conflict", noOp, err)
		}
	})

	t.Run("same path on different connections is still checked", func(t *testing.T) {
		noOp, err := ensureMoveTargetAvailable("/opt/source", "/opt/source", false, func(string) (bool, error) {
			return true, nil
		})
		if noOp || !errors.Is(err, ErrTargetExists) {
			t.Fatalf("noOp=%t err=%v, want conflict", noOp, err)
		}
	})

	t.Run("missing target is available", func(t *testing.T) {
		noOp, err := ensureMoveTargetAvailable("/opt/source", "/srv/source", true, func(string) (bool, error) {
			return false, nil
		})
		if err != nil || noOp {
			t.Fatalf("noOp=%t err=%v, want false nil", noOp, err)
		}
	})

	t.Run("stat error is preserved", func(t *testing.T) {
		statErr := errors.New("stat failed")
		noOp, err := ensureMoveTargetAvailable("/opt/source", "/srv/source", true, func(string) (bool, error) {
			return false, statErr
		})
		if noOp || !errors.Is(err, statErr) {
			t.Fatalf("noOp=%t err=%v, want stat error", noOp, err)
		}
	})
}

func TestStreamTextFileContentReports32KiBChunks(t *testing.T) {
	source := bytes.Repeat([]byte("x"), TextStreamChunkBytes*2+17)
	var chunks []TextReadChunkEvent

	err := streamTextFileContent(bytes.NewReader(source), "/tmp/demo.txt", "demo.txt", int64(len(source)), nil, func(event TextReadChunkEvent) {
		chunks = append(chunks, event)
	})
	if err != nil {
		t.Fatalf("streamTextFileContent returned error: %v", err)
	}
	if len(chunks) != 3 {
		t.Fatalf("chunk count=%d, want 3", len(chunks))
	}

	wantSizes := []int{TextStreamChunkBytes, TextStreamChunkBytes, 17}
	var got []byte
	for index, chunk := range chunks {
		if len(chunk.Data) != wantSizes[index] {
			t.Fatalf("chunk %d size=%d, want %d", index, len(chunk.Data), wantSizes[index])
		}
		wantOffset := int64(index * TextStreamChunkBytes)
		if chunk.OffsetBytes != wantOffset {
			t.Fatalf("chunk %d offset=%d, want %d", index, chunk.OffsetBytes, wantOffset)
		}
		got = append(got, chunk.Data...)
	}
	if !bytes.Equal(got, source) {
		t.Fatal("streamed chunks did not reconstruct source")
	}
}
