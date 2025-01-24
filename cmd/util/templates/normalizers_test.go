//go:build unit || !integration

package templates

import (
	"strings"
	"testing"
)

func TestLongDesc(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single line",
			input:    "A simple description",
			expected: "A simple description",
		},
		{
			name: "multi line with indentation",
			input: `
                This is a long description
                that spans multiple lines
                with indentation.
            `,
			expected: "This is a long description\nthat spans multiple lines\nwith indentation.",
		},
		{
			name: "multi line with empty lines",
			input: `
                First paragraph.

                Second paragraph.
                Still second paragraph.
            `,
			expected: "First paragraph.\n\nSecond paragraph.\nStill second paragraph.",
		},
		{
			name:     "trim whitespace",
			input:    "  \n  Description with spaces  \n  ",
			expected: "Description with spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LongDesc(tt.input)
			if got != tt.expected {
				t.Errorf("LongDesc() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExamples(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single example",
			input:    "# Simple example\ncommand arg1 arg2",
			expected: "  # Simple example\n  command arg1 arg2",
		},
		{
			name: "multiple examples",
			input: `# First example
command1 arg1
# Second example
command2 arg1 arg2`,
			expected: "  # First example\n  command1 arg1\n  # Second example\n  command2 arg1 arg2",
		},
		{
			name: "examples with empty lines",
			input: `# First example
command1 arg1

# Second example
command2 arg1`,
			expected: "  # First example\n  command1 arg1\n\n  # Second example\n  command2 arg1",
		},
		{
			name:     "preserve existing indentation",
			input:    "    # Indented example\n    command arg1",
			expected: "  # Indented example\n  command arg1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Examples(tt.input)
			if got != tt.expected {
				t.Errorf("Examples() =\n%q\nwant:\n%q", got, tt.expected)
			}
		})
	}
}

func TestNormalizerMethods(t *testing.T) {
	t.Run("heredoc", func(t *testing.T) {
		input := `
            First line
            Second line
        `
		n := normalizer{input}
		result := n.heredoc().string
		if strings.Contains(result, "            ") {
			t.Error("heredoc() did not remove leading whitespace")
		}
	})

	t.Run("trim", func(t *testing.T) {
		input := "  text with spaces  "
		n := normalizer{input}
		result := n.trim().string
		if result != "text with spaces" {
			t.Errorf("trim() = %q, want %q", result, "text with spaces")
		}
	})

	t.Run("indent", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "single line",
				input:    "text",
				expected: Indentation + "text",
			},
			{
				name:     "multiple lines",
				input:    "line1\nline2",
				expected: Indentation + "line1\n" + Indentation + "line2",
			},
			{
				name:     "empty lines preserved",
				input:    "line1\n\nline2",
				expected: Indentation + "line1\n\n" + Indentation + "line2",
			},
			{
				name:     "trim and indent",
				input:    "  line1  \n  line2  ",
				expected: Indentation + "line1\n" + Indentation + "line2",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				n := normalizer{tt.input}
				got := n.trim().indent().string
				if got != tt.expected {
					t.Errorf("indent() =\n%q\nwant:\n%q", got, tt.expected)
				}
			})
		}
	})
}
