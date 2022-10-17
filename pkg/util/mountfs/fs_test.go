package mountfs

import (
	"embed"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type mountFsSuite struct {
	suite.Suite
}

func TestMountFSSuite(t *testing.T) {
	suite.Run(t, new(mountFsSuite))
}

//go:embed *.go
var testFs embed.FS

//go:embed dir.go
var fileFs embed.FS

func getStandardMount() *MountDir {
	mount := New()
	mount.Mount("test", testFs)
	mount.Mount("more", fileFs)

	emptyMount := New()
	mount.Mount("empty", emptyMount)

	subMount := New()
	subMount.Mount("stuff", fileFs)
	mount.Mount("sub", subMount)

	mount.Mount("unmounted", testFs)
	mount.Unmount("unmounted")

	return mount
}

func (suite *mountFsSuite) TestMount() {
	mount := New()

	testCases := []struct {
		name  string
		mount string
		err   bool
	}{
		{"normal mount", "test", false},
		{"different mount", "more", false},
		{"same mount", "test", true},
		{"sub mount", filepath.Join("test", "more"), true},
		{"deep mount", filepath.Join("more", "test"), true},
	}

	for _, testCase := range testCases {
		suite.T().Run(testCase.name, func(t *testing.T) {
			err := mount.Mount(testCase.mount, testFs)
			require.True(t, (err != nil) == testCase.err)
		})
	}
}

func (suite *mountFsSuite) TestUnmount() {
	mount := getStandardMount()

	testCases := []struct {
		name  string
		mount string
		err   bool
	}{
		{"normal unmount", "test", false},
		{"different unmount", "more", false},
		{"same unmount", "test", true},
		{"random unmount", "booga", true},
		{"sub unmount", filepath.Join("sub", "stuff"), true},
		{"unmounted unmount", "unmounted", true},
	}

	for _, testCase := range testCases {
		suite.T().Run(testCase.name, func(t *testing.T) {
			err := mount.Unmount(testCase.mount)
			require.True(t, (err != nil) == testCase.err)
		})
	}
}

func (suite *mountFsSuite) TestEntries() {
	mount := getStandardMount()

	testCases := []struct {
		input    string
		expected []string
	}{
		{".", []string{"test", "more", "empty", "sub"}},
		{"test", []string{"dir.go", "direntry.go", "fs_test.go", "fs.go"}},
		{"more", []string{"dir.go"}},
		{"empty", []string{}},
		{"sub", []string{"stuff"}},
		{filepath.Join("sub", "stuff"), []string{"dir.go"}},
	}

	for _, testCase := range testCases {
		suite.T().Run(testCase.input, func(t *testing.T) {
			mounts, err := fs.ReadDir(mount, testCase.input)
			require.NoError(t, err)

			mountNames := []string{}
			for _, dirEntry := range mounts {
				mountNames = append(mountNames, dirEntry.Name())
			}
			require.ElementsMatch(t, mountNames, testCase.expected)
		})
	}

}
