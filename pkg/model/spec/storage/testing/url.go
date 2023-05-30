package storagetesting

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/url"
)

func URLDecodeStorage(t testing.TB, storage spec.Storage) *url.URLStorageSpec {
	out, err := url.Decode(storage)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
