package jobtransform

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/clone"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/rs/zerolog/log"
)

func RepoExistsOnIPFS(provider storage.StorageProvider) Transformer {
	return func(ctx context.Context, j *model.Job) (modified bool, err error) {
		inputs := j.Spec.Inputs
		modificationCount := 0

		for _, inputRepos := range inputs {
			var repoArray []string
			if inputRepos.StorageSource == model.StorageSourceRepoClone {
				repoArray = append(repoArray, inputRepos.Repo)
			}
			for _, url := range repoArray {
				repoCID, err := clone.RepoExistsOnIPFSGivenURL(ctx, url)
				log.Ctx(ctx).Error().Err(err).Msg("error checking whether repo exists")
				if err != nil {
					continue
				}

				inputs = clone.RemoveFromModelStorageSpec(inputs, url)

				inputs = append(inputs, model.StorageSpec{
					StorageSource: model.StorageSourceIPFS,
					CID:           repoCID,
					Path:          "/inputs",
				})
				modificationCount++
			}
		}

		j.Spec.Inputs = inputs
		return modificationCount != 0, nil
	}
}
