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
