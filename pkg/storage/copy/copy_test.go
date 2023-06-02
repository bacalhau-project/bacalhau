//go:build unit || !integration

package copy

import (
	"context"
	"strings"
	"testing"

	"github.com/c2h5oh/datasize"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/inline"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
	storage2 "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/testing"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/noop"
)

type copyOversizeTestCase struct {
	name     string
	specs    []*spec.Storage
	modified bool
}

const (
	maxSingle datasize.ByteSize = 10 * datasize.B
	maxTotal  datasize.ByteSize = 10 * datasize.B
)

var srcType spec.Storage
var dstType spec.Storage

func init() {
	var err error
	srcType, err = (&inline.InlineStorageSpec{URL: "https://example.com"}).AsSpec("TODO", "TODO")
	if err != nil {
		panic(err)
	}
	dstType, err = (&ipfs.IPFSStorageSpec{CID: storage2.TestCID1}).AsSpec("TODO", "TODO")
	if err != nil {
		panic(err)
	}
}

func makeUrlStorageSpec(t testing.TB, str string) *spec.Storage {
	out, err := (&inline.InlineStorageSpec{URL: str}).
		AsSpec("TODO", "TODO")
	require.NoError(t, err)
	return &out
}

func copyOversizeTestCases(t testing.TB) []copyOversizeTestCase {
	return []copyOversizeTestCase{
		{
			name:     "non specs are unchanged",
			specs:    []*spec.Storage{&dstType},
			modified: false,
		},
		{
			name:     "small specs are unchanged",
			specs:    []*spec.Storage{makeUrlStorageSpec(t, strings.Repeat("a", int(maxSingle)))},
			modified: false,
		},
		{
			name:     "large specs are changed",
			specs:    []*spec.Storage{makeUrlStorageSpec(t, strings.Repeat("a", int(maxSingle)+1))},
			modified: true,
		},
		{
			name: "multiple small specs below threshold are unchanged",
			specs: []*spec.Storage{
				makeUrlStorageSpec(t, strings.Repeat("a", int(maxTotal/2))),
				makeUrlStorageSpec(t, strings.Repeat("a", int(maxTotal/2))),
			},
			modified: false,
		},
		{
			name: "multiple small specs above threshold are changed",
			specs: []*spec.Storage{
				makeUrlStorageSpec(t, strings.Repeat("a", int(maxTotal/2)+1)),
				makeUrlStorageSpec(t, strings.Repeat("a", int(maxTotal/2))),
			},
			modified: true,
		},
	}
}

func preserveSlice[T any](slice []*T) []T {
	originals := make([]T, len(slice))
	for i, value := range slice {
		originals[i] = *value
	}
	return originals
}

func TestCopyOversize(t *testing.T) {
	for _, testCase := range copyOversizeTestCases(t) {
		t.Run(testCase.name, func(t *testing.T) {
			originals := preserveSlice(testCase.specs)

			didUpload := false
			noopStorage := noop.NewNoopStorageWithConfig(noop.StorageConfig{
				ExternalHooks: noop.StorageConfigExternalHooks{
					GetVolumeSize: func(ctx context.Context, volume spec.Storage) (uint64, error) {
						urlspec, err := inline.Decode(volume)
						if err != nil {
							return 0, err
						}
						return uint64(len(urlspec.URL)), nil
					},
					Upload: func(ctx context.Context, localPath string) (spec.Storage, error) {
						didUpload = true
						return (&ipfs.IPFSStorageSpec{CID: storage2.TestCID1}).AsSpec("TODO", "TODO")
					},
				},
			})

			provider := model.NewNoopProvider[cid.Cid, storage.Storage](noopStorage)
			modified, err := CopyOversize(
				context.Background(),
				provider,
				testCase.specs,
				inline.StorageType,
				ipfs.StorageType,
				maxSingle,
				maxTotal,
			)
			require.NoError(t, err)
			require.Equal(t, modified, testCase.modified)
			require.Equal(t, modified, didUpload)

			newSpecs := preserveSlice(testCase.specs)
			if modified {
				require.NotSubset(t, originals, newSpecs)
				require.NotSubset(t, newSpecs, originals)
			} else {
				require.Subset(t, originals, newSpecs)
				require.Subset(t, newSpecs, originals)
			}
		})
	}

}
