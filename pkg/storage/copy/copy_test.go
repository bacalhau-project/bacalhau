//go:build unit || !integration

package copy

import (
	"context"
	"strings"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/noop"
	"github.com/c2h5oh/datasize"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

type copyOversizeTestCase struct {
	name     string
	specs    []*models.InputSource
	modified bool
}

const (
	maxSingle datasize.ByteSize = 10 * datasize.B
	maxTotal  datasize.ByteSize = 10 * datasize.B
	srcType                     = models.StorageSourceInline
	dstType                     = models.StorageSourceIPFS
)

var copyOversizeTestCases = []copyOversizeTestCase{
	{
		name:     "non specs are unchanged",
		specs:    []*models.InputSource{{Source: &models.SpecConfig{Type: dstType}}},
		modified: false,
	},
	{
		name: "small specs are unchanged",
		specs: []*models.InputSource{{
			Source: &models.SpecConfig{
				Type: srcType,
				Params: map[string]interface{}{
					"URL": strings.Repeat("a", int(maxSingle)),
				},
			},
			Target: "/inputs",
		}},
		modified: false,
	},
	{
		name: "large specs are changed",
		specs: []*models.InputSource{{
			Source: &models.SpecConfig{
				Type: srcType,
				Params: map[string]interface{}{
					"URL": strings.Repeat("a", int(maxSingle)+1),
				},
			},
			Target: "/inputs",
		}},
		modified: true,
	},
	{
		name: "multiple small specs below threshold are unchanged",
		specs: []*models.InputSource{{
			Source: &models.SpecConfig{
				Type: srcType,
				Params: map[string]interface{}{
					"URL": strings.Repeat("a", int(maxTotal/2)),
				},
			},
			Target: "/inputs1",
		}, {
			Source: &models.SpecConfig{
				Type: srcType,
				Params: map[string]interface{}{
					"URL": strings.Repeat("a", int(maxTotal/2)),
				},
			},
			Target: "/inputs2",
		}},
		modified: false,
	},
	{
		name: "multiple small specs above threshold are changed",
		specs: []*models.InputSource{{
			Source: &models.SpecConfig{
				Type: srcType,
				Params: map[string]interface{}{
					"URL": strings.Repeat("a", int(maxTotal/2)+1),
				},
			},
			Target: "/inputs1",
		}, {
			Source: &models.SpecConfig{
				Type: srcType,
				Params: map[string]interface{}{
					"URL": strings.Repeat("a", int(maxTotal/2)),
				},
			},
			Target: "/inputs2",
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
			originals := preserveSlice(testCase.specs)

			didUpload := false
			noopStorage := noop.NewNoopStorageWithConfig(noop.StorageConfig{
				ExternalHooks: noop.StorageConfigExternalHooks{
					GetVolumeSize: func(ctx context.Context, volume models.InputSource) (uint64, error) {
						return uint64(len(volume.Source.Params["URL"].(string))), nil
					},
					Upload: func(ctx context.Context, localPath string) (models.SpecConfig, error) {
						didUpload = true
						return models.SpecConfig{Type: dstType}, nil
					},
				},
			})

			provider := provider.NewNoopProvider[storage.Storage](noopStorage)
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
			require.Equal(t, testCase.modified, modified)
			require.Equal(t, didUpload, modified)

			newSpecs := preserveSlice(testCase.specs)
			if modified {
				require.NotSubset(t, originals, newSpecs)
				require.NotSubset(t, newSpecs, originals)
			} else {
				require.Subset(t, originals, newSpecs)
				require.Subset(t, newSpecs, originals)
			}

			oldPaths := lo.Map(originals, func(s models.InputSource, _ int) string { return s.Target })
			newPaths := lo.Map(newSpecs, func(s models.InputSource, _ int) string { return s.Target })
			require.ElementsMatch(t, oldPaths, newPaths)
		})
	}

}
