package filecoinlotus

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multiaddr"
	"github.com/pelletier/go-toml/v2"
	"github.com/rs/zerolog/log"
)

type PublisherConfig struct {
	// Address of the miner to upload to
	MinerAddress string
	// How long the deal for the data should be created for
	StorageDuration time.Duration
	// Location of the Lotus configuration directory - either $LOTUS_PATH or ~/.lotus
	LotusDataDir string
	// Directory to use when uploading content to Lotus - optional
	LotusUploadDir string
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

	if config.MinerAddress == "" {
		return nil, errors.New("MinerAddress is required")
	}
	if config.StorageDuration == time.Duration(0) {
		return nil, errors.New("StorageDuration is required")
	}
	if config.LotusDataDir == "" {
		return nil, errors.New("LotusDataDir is required")
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

	miner, err := address.NewFromString(l.config.MinerAddress)
	if err != nil {
		return "", err
	}

	params, err := l.client.StateGetNetworkParams(ctx)
	if err != nil {
		return "", err
	}

	minerInfo, err := l.client.StateMinerInfo(ctx, miner, api.TipSetKey{})
	if err != nil {
		return "", err
	}

	// Note for the future - if we want to find miners, rather than be told them, then the lotus client currently:
	// * checks `StateMinerPower` to see if `HasMinPower` is true
	// * plays around with the 'ping' of each node, that is how long it took for the node to respond to `ClientQueryAsk`
	// * filters candidates based on whether their proposed amount is less than the budget
	// * select _n_ of the remaining miners

	power, err := l.client.StateMinerPower(ctx, miner, api.TipSetKey{})
	if err != nil {
		return "", err
	}
	if !power.HasMinPower {
		return "", fmt.Errorf("doesn't have min power")
	}

	ask, err := l.client.ClientQueryAsk(ctx, *minerInfo.PeerId, miner)
	if err != nil {
		return "", err
	}

	if ask.Response.MinPieceSize > dataSize.PieceSize {
		return "", fmt.Errorf("data too small")
	}
	if ask.Response.MaxPieceSize < dataSize.PieceSize {
		return "", fmt.Errorf("data too big")
	}

	epochPrice := big2.Div(big2.Mul(ask.Response.Price, big2.NewIntUnsigned(uint64(dataSize.PieceSize))), big2.NewInt(oneGibibyte))

	epochs := api.ChainEpoch(l.config.StorageDuration / (time.Duration(params.BlockDelaySecs) * time.Second))

	wallet, err := l.client.WalletDefaultAddress(ctx)
	if err != nil {
		return "", err
	}

	deal, err := l.client.ClientStartDeal(ctx, &api.StartDealParams{
		Data: &api.DataRef{
			TransferType: "graphsync", // storagemarket.TTGraphsync
			Root:         contentCid,
			PieceCid:     &dataSize.PieceCID,
			PieceSize:    dataSize.PieceSize.Unpadded(),
		},
		Wallet:            wallet,
		Miner:             miner,
		EpochPrice:        epochPrice,
		MinBlocksDuration: uint64(epochs),
	})
	if err != nil {
		return "", err
	}

	log.Ctx(ctx).Info().Stringer("cid", deal).Msg("Deal started")

	for {
		info, err := l.client.ClientGetDealInfo(ctx, *deal)
		if err != nil {
			return "", err
		}
		wanted := storagemarket.StorageDealCheckForAcceptance
		if info.State == wanted {
			return deal.String(), nil
		}

		if info.State == storagemarket.StorageDealFailing || info.State == storagemarket.StorageDealError {
			return "", fmt.Errorf("deal not accepted: %s", info.Message)
		}

		log.Ctx(ctx).Info().
			Stringer("cid", deal).
			Str("current", storagemarket.DealStates[info.State]).
			Str("expected", storagemarket.DealStates[wanted]).
			Msg("Deal not currently in expected state")
		time.Sleep(2 * time.Second)
	}
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
