//go:build unit || !integration

package bad

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

var errorTree = errors.Join(
	errors.Join(
		errors.New("1"),
		fmt.Errorf("test: %w", errors.New("2")),
	),
	errors.New("3"),
)

var errorTreeOutput = strings.Join([]string{
	"* 1",
	"* test: ",
	"\t* 2",
	"* 3",
	"",
}, "\n")

// Leaves should return only the errors in the tree with no children.
func TestLeaves(t *testing.T) {
	leaves := ToError(errorTree).Leaves()
	types := lo.Map(leaves, func(e Error, _ int) string { return e.Type })
	require.ElementsMatch(t, types, []string{"1", "2", "3"})

	require.Nil(t, (*Error)(nil).Leaves())
}

type testSingleUnwrap struct{}

func (testSingleUnwrap) Error() string { return "single" }
func (testSingleUnwrap) Unwrap() error { return nil }

type testMultiUnwrap struct{}

func (testMultiUnwrap) Error() string   { return "multi" }
func (testMultiUnwrap) Unwrap() []error { return []error{nil} }

// ToError should return nil for nil errors, the same object for Errors, and a
// new Error for native errors.
func TestToError(t *testing.T) {
	require.Nil(t, ToError(nil))

	x := Error{Type: "x"}
	require.Same(t, &x, ToError(&x))

	y := errors.New("y")
	require.Equal(t, "y", ToError(y).Type)

	z := ToError(testSingleUnwrap{})
	require.Equal(t, "single", z.Type)
	require.Empty(t, z.Errs)

	a := ToError(testMultiUnwrap{})
	require.Equal(t, "multi", a.Type)
	require.Empty(t, a.Errs)
}

// An error that just outputs the messages of its wrapped errors should not be
// converted into an Error with a Type.
func TestConcatenation(t *testing.T) {
	err := errors.Join(
		errors.New("test 1"),
		errors.New("test 2"),
	)
	require.Equal(t, "test 1\ntest 2", err.Error())

	badErr := ToError(err)
	require.Empty(t, badErr.Type)
	require.Len(t, badErr.Errs, 2)
}

// Output should display a tree of errors in a simple text view.
func TestErrorOutput(t *testing.T) {
	require.Empty(t, ToError(nil).Error())
	require.Equal(t, errorTreeOutput, ToError(errorTree).Error())
}

// Input should tag an error as being from an input source, should return nil on
// nil input, and should panic if more than one data item is passed.
func TestInput(t *testing.T) {
	require.Nil(t, Input(nil))

	err := Input(errors.New("err"))
	badErr, ok := (err).(*Error)
	require.True(t, ok)
	require.Equal(t, ErrSubjectInput, badErr.Subject)

	err = Input(errors.New("err"), "things")
	badErr, ok = (err).(*Error)
	require.True(t, ok)
	require.Equal(t, "things", badErr.Data)

	require.Panics(t, func() {
		Input(errors.New("err"), "1", "2")
	})
}
