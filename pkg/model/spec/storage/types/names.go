package types

import (
	"github.com/ipfs/go-cid"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/estuary"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/filecoin"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/git"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/gitlfs"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/inline"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/local"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/s3"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/url"
)

func StorageTypes() []cid.Cid {
	return []cid.Cid{
		estuary.StorageType,
		filecoin.StorageType,
		git.StorageType,
		gitlfs.StorageType,
		inline.StorageType,
		ipfs.StorageType,
		local.StorageType,
		s3.StorageType,
		url.StorageType,
	}
}

func StorageTypeNames() []string {
	return []string{
		estuary.StorageType.String(),
		filecoin.StorageType.String(),
		git.StorageType.String(),
		gitlfs.StorageType.String(),
		inline.StorageType.String(),
		ipfs.StorageType.String(),
		local.StorageType.String(),
		s3.StorageType.String(),
		url.StorageType.String(),
	}

}
