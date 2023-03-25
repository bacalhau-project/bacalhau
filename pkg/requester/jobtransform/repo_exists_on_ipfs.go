package jobtransform

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/clone"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

func RepoExistsOnIPFS(provider storage.StorageProvider) Transformer {
	return func(ctx context.Context, j *model.Job) (modified bool, err error) {
		inputs := j.Spec.Inputs
		ModificationCount := 0
		for _, inputRepos := range inputs {
			var repoArray []string
			if inputRepos.StorageSource == model.StorageSourceRepoClone {
				repoArray = append(repoArray, inputRepos.Repo)
			}
			for _, url := range repoArray {
				repoCID, _ := clone.RepoExistsOnIPFSGivenURL(url, ctx)
				// if err != nil {
				// 	fmt.Print(err)
				// }
				if repoCID != "" {
					inputs = clone.RemoveFromModelStorageSpec(inputs, url)

					inputs = append(inputs, model.StorageSpec{
						StorageSource: model.StorageSourceIPFS,
						CID:           repoCID,
						Path:          "/inputs",
					})
					ModificationCount++
				}
			}
		}
		j.Spec.Inputs = inputs
		return ModificationCount != 0, nil
	}
}
