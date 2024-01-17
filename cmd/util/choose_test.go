//go:build unit || !integration

package util

import (
	"bytes"
	"testing"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestNoChoices(t *testing.T) {
	cmd := &cobra.Command{}
	_, err := Choose(cmd, "No choices :(", []string{})
	require.Error(t, err)
}

func TestOneChoice(t *testing.T) {
	cmd := &cobra.Command{}
	choice := uuid.New()
	chosen, err := Choose(cmd, "One choice", []string{choice.String()})
	require.NoError(t, err)
	require.Equal(t, choice.String(), chosen)
}

func TestMultipleChoice(t *testing.T) {
	for _, testcase := range []struct {
		name, input, expected string
		errCheck              func(require.TestingT, error, ...any)
	}{
		{"first choice", "1\n", "one", require.NoError},
		{"last choice", "2\n", "two", require.NoError},
		{"some invalid choices", "3\n0\nbooga\n1\n", "one", require.NoError},
		{"only invalid choices", "3\n0\n", "", require.Error},
		{"no choice", "", "", require.Error},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			n, err := buf.WriteString(testcase.input)
			require.NoError(t, err)
			require.Len(t, testcase.input, n)

			cmd := &cobra.Command{}
			cmd.SetIn(buf)

			chosen, err := Choose(cmd, "Choose from these:", []string{"one", "two"})
			testcase.errCheck(t, err)
			require.Equal(t, testcase.expected, chosen)
		})
	}
}
