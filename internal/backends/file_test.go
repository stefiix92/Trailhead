package backends

import "testing"

func TestLooksErrorLine(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"error: boom", true},
		{"ERROR boom", true},
		{"exception raised", true},
		{"panic: oh no", true},
		{"fatal: goodbye", true},
		{"err: short", true},
		{"errors are plural", true},
		{"it errored yesterday", true},
		{"terror is scary", false},
		{"ok line", false},
		{"no issues here", false},
		{"erroredly", false},
	}

	for _, tc := range cases {
		got := looksErrorLine(tc.in)
		if got != tc.want {
			t.Fatalf("looksErrorLine(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

