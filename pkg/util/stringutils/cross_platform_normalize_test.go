//go:build unit || !integration

package stringutils

import "testing"

func TestUpdateLineEndingsForPlatform(t *testing.T) {
	// Test cases - last test case has spaces and line endings (which are stripped in comparison)
	cases := []struct {
		name, input, platform, want string
	}{
		{"Unix to Unix", "Hello\nWorld\n", "linux", "Hello\nWorld\n"},
		{"Windows to Unix", "Hello\r\nWorld\r\n", "linux", "Hello\nWorld\n"},
		{"Mixed to Unix", "Hello\nWorld\r\n", "linux", "Hello\nWorld\n"},
		{"Unix to Windows", "Hello\nWorld\n", "windows", "Hello\r\nWorld\r\n"},
		{"Windows to Windows", "Hello\r\nWorld\r\n", "windows", "Hello\r\nWorld\r\n"},
		{"Mixed to Windows", "Hello\nWorld\r\n", "windows", "Hello\r\nWorld\r\n"},
		{"Example string linux", "Create a job from a file or from stdin.\n\n JSON and YAML formats are accepted.", "linux", "Create a job from a file or from stdin.\n\n JSON and YAML formats are accepted."},
		{"Example string windows", "Create a job from a file or from stdin.\r\n\r\n JSON and YAML formats are accepted.", "windows", "Create a job from a file or from stdin.\r\n\r\n JSON and YAML formats are accepted."},
		{"Blanks and line endings", `


`, "unix", `


`},
	}

	// Run the test cases
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := crossPlatformNormalizeLineEndings(tc.input, tc.platform)
			if got != tc.want {
				t.Errorf("got %q; want %q", got, tc.want)
			}
		})
	}
}
