package filecoinlotus

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/ipfs/car"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus/api"
	"github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus/api/storagemarket"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/go-address"
	big2 "github.com/filecoin-project/go-state-types/big"
	"github.com/hashicorp/go-multierror"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multiaddr"
	"github.com/pelletier/go-toml/v2"
	"github.com/rs/zerolog/log"
)

type PublisherConfig struct {
	// How long the deal for the data should be created for
	StorageDuration time.Duration
	// Location of the Lotus configuration directory - either $LOTUS_PATH or ~/.lotus
	LotusDataDir string
	// Directory to use when uploading content to Lotus - optional
	LotusUploadDir string
	// How close miner should be when selecting the cheapest
	MaximumPing time.Duration
}

type Publisher struct {
	stateResolver *job.StateResolver
	config        PublisherConfig
	client        api.Client
}

func NewFilecoinLotusPublisher(
	ctx context.Context,
	cm *system.CleanupManager,
	resolver *job.StateResolver,
	config PublisherConfig,
) (*Publisher, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publisher/filecoin_lotus/NewFilecoinLotusPublisher")
	defer span.End()

	if config.StorageDuration == time.Duration(0) {
		return nil, errors.New("StorageDuration is required")
	}
	if config.LotusDataDir == "" {
		return nil, errors.New("LotusDataDir is required")
	}
	if config.MaximumPing == time.Duration(0) {
		return nil, errors.New("MaximumPing is required")
	}

	tokenFile := filepath.Join(config.LotusDataDir, "token")
	token, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("unable to open token file %s: %w", tokenFile, err)
	}

	configFile := filepath.Join(config.LotusDataDir, "config.toml")
	hostname, err := fetchHostnameFromConfig(configFile)
	if err != nil {
		return nil, err
	}

	log.Ctx(ctx).Debug().Str("hostname", hostname).Msg("Building Lotus client")

	client, err := api.NewClient(ctx, hostname, string(token))
	if err != nil {
		return nil, err
	}
	cm.RegisterCallback(client.Close)

	return &Publisher{
		stateResolver: resolver,
		config:        config,
		client:        client,
	}, nil
}

func (l *Publisher) IsInstalled(ctx context.Context) (bool, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publisher/filecoin_lotus/IsInstalled")
	defer span.End()

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
	ctx, span := system.GetTracer().Start(ctx, "pkg/publisher/filecoin_lotus/PublishShardResult")
	defer span.End()

	log.Ctx(ctx).Debug().
		Stringer("shard", shard).
		Str("host", hostID).
		Str("shardResultPath", shardResultPath).
		Msg("Uploading results folder to filecoin lotus")

	tarFile, err := l.carResultsDir(ctx, shardResultPath)
	if err != nil {
		return model.StorageSpec{}, err
	}

	contentCid, err := l.importData(ctx, tarFile)
	if err != nil {
		return model.StorageSpec{}, err
	}

	dealCid, err := l.createDeal(ctx, contentCid)
	if err != nil {
		return model.StorageSpec{}, err
	}

	return model.StorageSpec{
		Name:          fmt.Sprintf("job-%s-shard-%d-host-%s", shard.Job.ID, shard.Index, hostID),
		StorageSource: model.StorageSourceFilecoin,
		CID:           contentCid.String(),
		Metadata: map[string]string{
			"deal_cid": dealCid,
		},
	}, nil
}

func (l *Publisher) ComposeResultReferences(
	ctx context.Context,
	jobID string,
) ([]model.StorageSpec, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publisher/filecoin_lotus/ComposeResultReferences")
	defer span.End()

	system.AddJobIDFromBaggageToSpan(ctx, span)

	var results []model.StorageSpec
	shardResults, err := l.stateResolver.GetResults(ctx, jobID)
	if err != nil {
		return results, err
	}
	for _, shardResult := range shardResults {
		results = append(results, shardResult.Results)
	}
	return results, nil
}

func (l *Publisher) carResultsDir(ctx context.Context, resultsDir string) (string, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publisher/filecoin_lotus/carResultsDir")
	defer span.End()

	tempDir, err := os.MkdirTemp(l.config.LotusUploadDir, "bacalhau-filecoin-lotus-*")
	if err != nil {
		return "", err
	}
	tempFile := filepath.Join(tempDir, "results.tar")

	if _, err := car.CreateCar(ctx, resultsDir, tempFile, 1); err != nil {
		return "", err
	}

	return tempFile, nil
}

func (l *Publisher) importData(ctx context.Context, filePath string) (cid.Cid, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publisher/filecoin_lotus/importData")
	defer span.End()

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
	ctx, span := system.GetTracer().Start(ctx, "pkg/publisher/filecoin_lotus/createDeal")
	defer span.End()

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
				if info.State == wanted {
					return nil
				}

				if info.State == storagemarket.StorageDealFailing || info.State == storagemarket.StorageDealError {
					return fmt.Errorf("deal not accepted: %s", info.Message)
				}
				currentState = info.State
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

func fetchHostnameFromConfig(file string) (string, error) {
	unparsedConfig, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("unable to open config file %s: %w", file, err)
	}
	var config struct {
		API struct {
			ListenAddress string
		}
	}
	if err := toml.Unmarshal(unparsedConfig, &config); err != nil { //nolint:govet
		return "", fmt.Errorf("unable to parse config file %s: %w", file, err)
	}

	addr, err := multiaddr.NewMultiaddr(config.API.ListenAddress)
	if err != nil {
		return "", fmt.Errorf("unable to parse ListenAddress in config file %s: %w", file, err)
	}

	var host, port string
	multiaddr.SplitFunc(addr, func(component multiaddr.Component) bool {
		switch component.Protocol().Code {
		case multiaddr.P_IP4:
			h := component.Value()
			if h == "0.0.0.0" {
				h = "localhost"
			}
			host = h
		case multiaddr.P_TCP:
			port = component.Value()
		}

		return host != "" && port != ""
	})

	if host == "" {
		return "", fmt.Errorf("unable to parse host from ListenAddress in config file %s", file)
	}
	if port == "" {
		return "", fmt.Errorf("unable to parse port from ListenAddress in config file %s", file)
	}

	return fmt.Sprintf("%s:%s", host, port), nil
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
