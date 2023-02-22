package copy

import (
	"context"
	"strings"
	"testing"

	"github.com/c2h5oh/datasize"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/noop"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
)

type copyOversizeTestCase struct {
	name     string
	specs    []*model.StorageSpec
	modified bool
}

const (
	maxSingle datasize.ByteSize       = 10 * datasize.B
	maxTotal  datasize.ByteSize       = 10 * datasize.B
	srcType   model.StorageSourceType = model.StorageSourceInline
	dstType   model.StorageSourceType = model.StorageSourceIPFS
)

var copyOversizeTestCases = []copyOversizeTestCase{
	{
		name:     "non specs are unchanged",
		specs:    []*model.StorageSpec{{StorageSource: dstType}},
		modified: false,
	},
	{
		name: "small specs are unchanged",
		specs: []*model.StorageSpec{{
			StorageSource: srcType,
			URL:           strings.Repeat("a", int(maxSingle)),
		}},
		modified: false,
	},
	{
		name: "large specs are changed",
		specs: []*model.StorageSpec{{
			StorageSource: srcType,
			URL:           strings.Repeat("a", int(maxSingle)+1),
		}},
		modified: true,
	},
	{
		name: "multiple small specs below threshold are unchanged",
		specs: []*model.StorageSpec{{
			StorageSource: srcType,
			URL:           strings.Repeat("a", int(maxTotal/2)),
		}, {
			StorageSource: srcType,
			URL:           strings.Repeat("a", int(maxTotal/2)),
		}},
		modified: false,
	},
	{
		name: "multiple small specs above threshold are changed",
		specs: []*model.StorageSpec{{
			StorageSource: srcType,
			URL:           strings.Repeat("a", int(maxTotal/2)+1),
		}, {
			StorageSource: srcType,
			URL:           strings.Repeat("a", int(maxTotal/2)),
		}},
		modified: true,
	},
}

func preserveSlice[T any](slice []*T) []T {
	originals := make([]T, len(slice))
	for i, value := range slice {
		originals[i] = *value
	}
	return originals
}

func TestCopyOversize(t *testing.T) {
	for _, testCase := range copyOversizeTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			cm := system.NewCleanupManager()
			t.Cleanup(func() {
				cm.Cleanup(context.Background())
			})

			originals := preserveSlice(testCase.specs)

			didUpload := false
			noopStorage, err := noop.NewNoopStorageWithConfig(context.Background(), cm, noop.StorageConfig{
				ExternalHooks: noop.StorageConfigExternalHooks{
					GetVolumeSize: func(ctx context.Context, volume model.StorageSpec) (uint64, error) {
						return uint64(len(volume.URL)), nil
					},
					Upload: func(ctx context.Context, localPath string) (model.StorageSpec, error) {
						didUpload = true
						return model.StorageSpec{StorageSource: dstType}, nil
					},
				},
			})
			require.NoError(t, err)

			provider := model.NewNoopProvider[model.StorageSourceType, storage.Storage](noopStorage)
			modified, err := CopyOversize(
				context.Background(),
				provider,
				testCase.specs,
				srcType,
				dstType,
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
