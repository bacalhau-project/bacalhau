package jobtransform

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/clone"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/git"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

func RepoExistsOnIPFS(provider storage.StorageProvider) Transformer {
	return func(ctx context.Context, j *model.Job) (modified bool, err error) {
		inputs := j.Spec.Inputs
		modificationCount := 0

		for _, inputRepos := range j.Spec.Inputs {
			var repoArray []string
			if inputRepos.Schema == git.StorageType {
				gitspec, err := git.Decode(inputRepos)
				if err != nil {
					return false, err
				}
				repoArray = append(repoArray, gitspec.Repo)
			}
			for _, url := range repoArray {
				repoCID, err := clone.RepoExistsOnIPFSGivenURL(ctx, url)
				if err != nil {
					log.Ctx(ctx).Error().Err(err).Msg("error checking whether repo exists")
					// TODO we want to return this, right? As a user I wouldn't want a half transformed spec
					return false, err
				}

				// TODO(forrest): I have no idea what the the logic in this method is trying to do
				// this looks broken. It was previously overriding the inputs then appending to it??
				// I have attempted to correct it but unsure what initial intention was here.
				// NEEDS REVIEW
				modifiedInputs, err := clone.RemoveFromModelStorageSpec(inputs, url)
				if err != nil {
					return false, err
				}
				inputs = append(inputs, modifiedInputs...)

				c, err := cid.Decode(repoCID)
				if err != nil {
					return false, err
				}

				ipfsspec, err := (&ipfs.IPFSStorageSpec{
					CID: c,
				}).AsSpec("TODO", "/inputs")

				inputs = append(inputs, ipfsspec)
				modificationCount++
			}
		}

		// modify the spec inputs
		j.Spec.Inputs = inputs
		return modificationCount != 0, nil
	}
}
