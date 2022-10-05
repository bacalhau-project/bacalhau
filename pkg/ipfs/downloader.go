package ipfs

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type IPFSDownloadSettings struct {
	TimeoutSecs    int
	OutputDir      string
	IPFSSwarmAddrs string
}

const DefaultIPFSTimeout time.Duration = 5 * time.Minute

func NewIPFSDownloadSettings() *IPFSDownloadSettings {
	return &IPFSDownloadSettings{
		TimeoutSecs:    int(DefaultIPFSTimeout.Seconds()),
		OutputDir:      ".",
		IPFSSwarmAddrs: "",
	}
}

// * make a temp dir
// * download all cids into temp dir
// * ensure top level output dir exists
// * iterate over each shard
// * make new folder for shard logs
// * copy stdout, stderr, exitCode
// * append stdout, stderr to global log
// * iterate over each output volume
// * make new folder for output volume
// * iterate over each shard and merge files in output folder to results dir
func DownloadJob( //nolint:funlen,gocyclo
	ctx context.Context,
	cm *system.CleanupManager,
	j *model.Job,
	results []model.StorageSpec,
	settings IPFSDownloadSettings,
) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.DownloadJob")
	defer span.End()

	if len(results) == 0 {
		log.Ctx(ctx).Debug().Msg("No results to download")
		return nil
	}

	switch system.GetEnvironment() {
	case system.EnvironmentProd:
		settings.IPFSSwarmAddrs = strings.Join(system.Envs[system.Production].IPFSSwarmAddresses, ",")
	case system.EnvironmentDev:
		// TODO: add more dev swarm addresses?
		if os.Getenv("BACALHAU_IPFS_SWARM_ADDRESSES") != "" {
			settings.IPFSSwarmAddrs = os.Getenv("BACALHAU_IPFS_SWARM_ADDRESSES")
		}
	case system.EnvironmentStaging:
		log.Ctx(ctx).Warn().Msg("Staging environment has no IPFS swarm addresses attached")
	}

	// NOTE: we have to spin up a temporary IPFS node as we don't
	// generally have direct access to a remote node's API server.
	n, err := spinUpIPFSNode(ctx, cm, settings.IPFSSwarmAddrs)
	if err != nil {
		return err
	}

	err = loopOverResults(ctx, n, results, settings, j)
	if err != nil {
		return err
	}

	return nil
}

func loopOverResults(ctx context.Context,
	n *Node,
	results []model.StorageSpec,
	settings IPFSDownloadSettings,
	j *model.Job) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.loopingOverResults")
	defer span.End()

	log.Ctx(ctx).Debug().Msg("Connecting client to new IPFS node...")
	cl, err := n.Client()
	if err != nil {
		return err
	}

	finalOutputDirAbs, err := filepath.Abs(settings.OutputDir)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Failed to get absolute path for output dir: %s", err)
		return err
	}

	// loop over each result directory
	// each result is a storage spec representing a single shards output
	// it's "name" and "path" is named after the shard index
	// so we write the shard output to our scratch folder
	// and then merge each outout volume into the global results
	log.Ctx(ctx).Info().Msgf("Found %d result shards, downloading to temporary folder.", len(results))

	// we move all the contents of the output volume to the global results dir
	// for this output volume
	// find $SOURCE_DIR -name '*' -type f -exec mv -f {} $TARGET_DIR \;
	// append all stdout and stderr to a global concatenated log
	// make a directory for the individual shard logs
	// move the stdout, stderr, and exit code to the shard results dir
	for _, result := range results {
		shardDownloadDir := filepath.Join(finalOutputDirAbs, result.Name)
		err := fetchResult(ctx, result, cl, shardDownloadDir, settings.TimeoutSecs)
		if err != nil {
			return err
		}
	}
	return nil
}

func spinUpIPFSNode(ctx context.Context,
	cm *system.CleanupManager,
	ipfsSwarmAddrs string) (*Node, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.DownloadJob.SpinningUpIPFS")
	defer span.End()

	log.Ctx(ctx).Debug().Msg("Spinning up IPFS node...")
	n, err := NewNode(ctx, cm, strings.Split(ipfsSwarmAddrs, ","))
	if err != nil {
		return nil, err
	}
	return n, nil
}

func fetchResult(ctx context.Context,
	result model.StorageSpec,
	cl *Client,
	shardDownloadDir string,
	timeoutSecs int) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.fetchingResult")
	defer span.End()

	err := func() error {
		log.Ctx(ctx).Debug().Msgf("Downloading result CID %s '%s' to '%s'...", result.Name, result.CID, shardDownloadDir)

		innerCtx, cancel := context.WithDeadline(ctx,
			time.Now().Add(time.Second*time.Duration(timeoutSecs)))
		defer cancel()

		return cl.Get(innerCtx, result.CID, shardDownloadDir)
	}()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Ctx(ctx).Error().Msg("Timed out while downloading result.")
		}

		return err
	}
	return nil
}
