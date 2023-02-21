package filecoinlotus

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/ipfs/car"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus/api"
	"github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus/api/storagemarket"
	"github.com/filecoin-project/bacalhau/pkg/storage/util"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/go-address"
	big2 "github.com/filecoin-project/go-state-types/big"
	"github.com/hashicorp/go-multierror"
	"github.com/ipfs/go-cid"
	"github.com/rs/zerolog/log"
)

type PublisherConfig struct {
	// How long the deal for the data should be created for
	StorageDuration time.Duration
	// Location of the Lotus configuration directory - either $LOTUS_PATH or ~/.lotus
	PathDir string
	// Directory to use when uploading content to Lotus - optional
	UploadDir string
	// How close miner should be when selecting the cheapest
	MaximumPing time.Duration
}

type Publisher struct {
	config PublisherConfig
	client api.Client
}

func NewPublisher(
	ctx context.Context,
	cm *system.CleanupManager,
	config PublisherConfig,
) (*Publisher, error) {
	if config.StorageDuration == time.Duration(0) {
		return nil, errors.New("StorageDuration is required")
	}
	if config.PathDir == "" {
		return nil, errors.New("PathDir is required")
	}
	if config.MaximumPing == time.Duration(0) {
		return nil, errors.New("MaximumPing is required")
	}

	client, err := api.NewClientFromConfigDir(ctx, config.PathDir)
	if err != nil {
		return nil, err
	}
	cm.RegisterCallback(client.Close)

	return newPublisher(config, client), nil
}

func newPublisher(
	config PublisherConfig,
	client api.Client,
) *Publisher {
	return &Publisher{
		config: config,
		client: client,
	}
}

func (l *Publisher) IsInstalled(ctx context.Context) (bool, error) {
	if _, err := l.client.Version(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (l *Publisher) PublishShardResult(
	ctx context.Context,
	shard model.JobShard,
	hostID string,
	shardResultPath string,
) (model.StorageSpec, error) {
	log.Ctx(ctx).Debug().
		Stringer("shard", shard).
		Str("host", hostID).
		Str("shardResultPath", shardResultPath).
		Msg("Uploading results folder to filecoin lotus")

	carFile, err := l.carResultsDir(ctx, shardResultPath)
	if err != nil {
		return model.StorageSpec{}, err
	}

	contentCid, err := l.importData(ctx, carFile)
	if err != nil {
		return model.StorageSpec{}, err
	}

	dealCid, err := l.createDeal(ctx, contentCid)
	if err != nil {
		return model.StorageSpec{}, err
	}

	spec := job.GetPublishedStorageSpec(shard, model.StorageSourceFilecoin, hostID, contentCid.String())
	spec.Metadata["deal_cid"] = dealCid
	return spec, nil
}

func (l *Publisher) carResultsDir(ctx context.Context, resultsDir string) (string, error) {
	tempFile, err := os.CreateTemp(l.config.UploadDir, "results-*.car")
	if err != nil {
		return "", err
	}

	// Temporary files will have 0600 as their permissions, which could cause issues when sharing with a Lotus node
	// running inside a container.
	if err := tempFile.Chmod(util.OS_ALL_RW); err != nil { //nolint:govet
		return "", err
	}

	// Just need the filename
	if err := tempFile.Close(); err != nil {
		return "", err
	}

	if _, err := car.CreateCar(ctx, resultsDir, tempFile.Name(), 1); err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

func (l *Publisher) importData(ctx context.Context, filePath string) (cid.Cid, error) {
	res, err := l.client.ClientImport(ctx, api.FileRef{
		Path:  filePath,
		IsCAR: true,
	})
	if err != nil {
		return cid.Cid{}, err
	}
	return res.Root, nil
}

func (l *Publisher) createDeal(ctx context.Context, contentCid cid.Cid) (string, error) {
	dataSize, err := l.client.ClientDealPieceCID(ctx, contentCid)
	if err != nil {
		return "", err
	}

	params, err := l.client.StateGetNetworkParams(ctx)
	if err != nil {
		return "", err
	}

	epochs := api.ChainEpoch(l.config.StorageDuration / (time.Duration(params.BlockDelaySecs) * time.Second))

	wallet, err := l.client.WalletDefaultAddress(ctx)
	if err != nil {
		return "", err
	}

	miners, err := l.client.StateListMiners(ctx, api.TipSetKey{})
	if err != nil {
		return "", err
	}

	log.Ctx(ctx).Debug().Int("count", len(miners)).Msg("Initial list of miners")

	asks, errs := throttledMap(miners, func(miner address.Address) (*ask, error) {
		return l.queryMiner(ctx, dataSize, miner)
	}, parallelMinerQueries)
	if len(asks) == 0 {
		log.Ctx(ctx).
			Err(multierror.Append(nil, errs...)).
			Msg("Couldn't find a miner")
		return "", fmt.Errorf("unable to find a miner")
	}

	cheapest := asks[0]
	for _, a := range asks {
		if a.epochPrice.LessThan(cheapest.epochPrice) {
			cheapest = a
		}
	}

	deal, err := l.client.ClientStartDeal(ctx, &api.StartDealParams{
		Data: &api.DataRef{
			TransferType: "graphsync", // storagemarket.TTGraphsync
			Root:         contentCid,
			PieceCid:     &dataSize.PieceCID,
			PieceSize:    dataSize.PieceSize.Unpadded(),
		},
		Wallet:            wallet,
		Miner:             cheapest.miner,
		EpochPrice:        cheapest.epochPrice,
		MinBlocksDuration: uint64(epochs),
	})
	if err != nil {
		return "", err
	}

	log.Ctx(ctx).Info().Stringer("cid", deal).Msg("Deal started")

	if err := l.waitUntilDealIsReady(ctx, deal); err != nil {
		return "", err
	}

	return deal.String(), nil
}

func (l *Publisher) waitUntilDealIsReady(ctx context.Context, deal *cid.Cid) error {
	// The go-jsonrpc library that the `client` uses relies on the context to know when to stop writing to the info channel
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	infoChan, err := l.client.ClientGetDealUpdates(ctx)
	if err != nil {
		return err
	}

	t := time.NewTicker(3 * time.Second)
	defer t.Stop()

	// The documentation recommends that, at least for lite nodes, we should wait until the deal's state is `StorageDealActive`.
	// This can take a long time, an hour or so, with the test image.
	// Additional states after `StorageDealCheckForAcceptance` are:
	// * `StorageDealAwaitingPreCommit` is reached once the sector available for sealing - a.k.a. no more data allowed
	// * `StorageDealSealing` is reached after PreCommit has happened (150 epochs?)
	// * `StorageDealActive` - sector has been sealed and everything is ready
	var currentState storagemarket.StorageDealStatus
	wanted := storagemarket.StorageDealCheckForAcceptance
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case info := <-infoChan:
			if deal.Equals(info.ProposalCid) {
				currentState = info.State

				if currentState == wanted {
					log.Ctx(ctx).Info().
						Stringer("deal", deal).
						Str("current", storagemarket.DealStates[currentState]).
						Str("expected", storagemarket.DealStates[wanted]).
						Msg("Deal in expected state")
					return nil
				}

				if currentState == storagemarket.StorageDealFailing || currentState == storagemarket.StorageDealError {
					return fmt.Errorf("deal not accepted: %s", info.Message)
				}
			}
		case <-t.C:
			log.Ctx(ctx).Info().
				Stringer("deal", deal).
				Str("current", storagemarket.DealStates[currentState]).
				Str("expected", storagemarket.DealStates[wanted]).
				Msg("Deal not currently in expected state")
		}
	}
}

func (l *Publisher) queryMiner(ctx context.Context, dataSize api.DataCIDSize, miner address.Address) (*ask, error) {
	minerInfo, err := l.client.StateMinerInfo(ctx, miner, api.TipSetKey{})
	if err != nil {
		return nil, fmt.Errorf("failed to get miner %s info: %w", miner, err)
	}

	power, err := l.client.StateMinerPower(ctx, miner, api.TipSetKey{})
	if err != nil {
		return nil, fmt.Errorf("failed to get miner %s power: %w", miner, err)
	}
	if !power.HasMinPower {
		return nil, fmt.Errorf("miner %s doesn't have min power", miner)
	}

	start := time.Now()
	query, err := l.client.ClientQueryAsk(ctx, *minerInfo.PeerId, miner)
	if err != nil {
		return nil, fmt.Errorf("failed to query miner %s: %w", miner, err)
	}
	ping := time.Since(start)

	if ping > l.config.MaximumPing {
		return nil, fmt.Errorf("ping for miner %s (%s) is too large", miner, ping)
	}

	if query.Response.MinPieceSize > dataSize.PieceSize {
		return nil, fmt.Errorf("data size (%v) is too small for miner %s (%v)", dataSize.PieceSize, miner, query.Response.MinPieceSize)
	}
	if query.Response.MaxPieceSize < dataSize.PieceSize {
		return nil, fmt.Errorf("data size (%v) is too big for miner %s (%v)", dataSize.PieceSize, miner, query.Response.MaxPieceSize)
	}

	epochPrice := big2.Div(big2.Mul(query.Response.Price, big2.NewIntUnsigned(uint64(dataSize.PieceSize))), big2.NewInt(oneGibibyte))

	return &ask{
		miner:      miner,
		epochPrice: epochPrice,
	}, nil
}

type ask struct {
	miner      address.Address
	epochPrice big2.Int
}

var _ publisher.Publisher = &Publisher{}

const oneGibibyte = 1 << 30
const parallelMinerQueries = 50

func throttledMap[T any, V comparable](ts []T, f func(T) (V, error), concurrent int) ([]V, []error) {
	throttle := make(chan struct{}, concurrent)
	mu := sync.Mutex{}
	var wg sync.WaitGroup

	var errs []error
	var vs []V

	var empty V

	for _, t := range ts {
		t := t
		wg.Add(1)
		throttle <- struct{}{}
		go func() {
			defer func() {
				<-throttle
			}()
			defer wg.Done()

			v, err := f(t)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
			} else if v != empty {
				vs = append(vs, v)
			}
		}()
	}

	wg.Wait()

	return vs, errs
}
