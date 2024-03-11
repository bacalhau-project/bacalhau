//go:build unit || !integration

package string

import "testing"

func TestUpdateLineEndingsForPlatform(t *testing.T) {
	// Test cases - last test case has spaces and line endings (which are stripped in comparison)
	cases := []struct {
		name, input, platform, want string
	}{
		{"Unix to Unix", "Hello\nWorld\n", "unix", "Hello\nWorld\n"},
		{"Windows to Unix", "Hello\r\nWorld\r\n", "unix", "Hello\nWorld\n"},
		{"Mixed to Unix", "Hello\nWorld\r\n", "unix", "Hello\nWorld\n"},
		{"Unix to Windows", "Hello\nWorld\n", "windows", "Hello\r\nWorld\r\n"},
		{"Windows to Windows", "Hello\r\nWorld\r\n", "windows", "Hello\r\nWorld\r\n"},
		{"Mixed to Windows", "Hello\nWorld\r\n", "windows", "Hello\r\nWorld\r\n"},
		{"Blanks and line endings", `


`, "unix", `


`},
	}

	// Run the test cases
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := crossPlatformNormalizeLineEndings(tc.input, tc.platform)
			want := crossPlatformNormalizeLineEndings(tc.want, tc.platform)
			if got != want {
				t.Errorf("got %q; want %q", got, want)
			}
		})
	}
}
