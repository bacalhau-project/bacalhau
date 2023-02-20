//nolint:gochecknoinits,stylecheck // Most of the code in this package is copied with hope to use the upstream version
package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus/api/retrievalmarket"
	"github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus/api/storagemarket"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/telemetry"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	abi2 "github.com/filecoin-project/go-state-types/abi"
	big2 "github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/builtin/v9/miner"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/pelletier/go-toml/v2"
	"github.com/rs/zerolog/log"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

func NewClient(ctx context.Context, host string, token string) (Client, error) {
	log.Ctx(ctx).Debug().Str("hostname", host).Msg("Building Lotus client")

	headers := http.Header{"Authorization": []string{fmt.Sprintf("Bearer %s", token)}}

	u := url.URL{
		Scheme: "ws",
		Host:   host,
		Path:   "/rpc/v1",
	}

	var client api
	closer, err := jsonrpc.NewMergeClient(ctx, u.String(), "Filecoin", []interface{}{&client.internal}, headers)
	if err != nil {
		return nil, fmt.Errorf("unable to create client to %s: %w", host, err)
	}
	client.close = closer
	client.hostname = host

	return &client, nil
}

func NewClientFromConfigDir(ctx context.Context, dir string) (Client, error) {
	tokenFile := filepath.Join(dir, "token")
	token, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("unable to open token file %s: %w", tokenFile, err)
	}

	configFile := filepath.Join(dir, "config.toml")
	hostname, err := fetchHostnameFromConfig(configFile)
	if err != nil {
		return nil, err
	}

	client, err := NewClient(ctx, hostname, string(token))
	if err != nil {
		return nil, err
	}

	return client, nil
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

// Importing the Lotus API causes dependency conflicts between this project and Lotus
// So 're-implement' the API here to avoid the conflicts
// https://github.com/filecoin-project/lotus/blob/master/api/api_full.go

type Client interface {
	ClientDealPieceCID(context.Context, cid.Cid) (DataCIDSize, error)
	ClientExport(context.Context, ExportRef, FileRef) error
	ClientGetDealUpdates(ctx context.Context) (<-chan DealInfo, error)
	ClientListImports(context.Context) ([]Import, error)
	ClientImport(context.Context, FileRef) (*ImportRes, error)
	ClientQueryAsk(context.Context, peer.ID, address.Address) (*StorageAsk, error)
	ClientStartDeal(context.Context, *StartDealParams) (*cid.Cid, error)
	StateGetNetworkParams(context.Context) (*NetworkParams, error)
	StateListMiners(context.Context, TipSetKey) ([]address.Address, error)
	StateMinerInfo(context.Context, address.Address, TipSetKey) (MinerInfo, error)
	StateMinerPower(context.Context, address.Address, TipSetKey) (*MinerPower, error)
	Version(context.Context) (APIVersion, error)
	WalletDefaultAddress(context.Context) (address.Address, error)

	Close() error
}

var _ Client = &api{}

type api struct {
	internal struct {
		ClientDealPieceCID    func(context.Context, cid.Cid) (DataCIDSize, error)
		ClientExport          func(context.Context, ExportRef, FileRef) error
		ClientGetDealUpdates  func(ctx context.Context) (<-chan DealInfo, error)
		ClientListImports     func(context.Context) ([]Import, error)
		ClientImport          func(context.Context, FileRef) (*ImportRes, error)
		ClientQueryAsk        func(context.Context, peer.ID, address.Address) (*StorageAsk, error)
		ClientStartDeal       func(context.Context, *StartDealParams) (*cid.Cid, error)
		StateGetNetworkParams func(context.Context) (*NetworkParams, error)
		StateListMiners       func(context.Context, TipSetKey) ([]address.Address, error)
		StateMinerInfo        func(context.Context, address.Address, TipSetKey) (MinerInfo, error)
		StateMinerPower       func(context.Context, address.Address, TipSetKey) (*MinerPower, error)
		Version               func(context.Context) (APIVersion, error)
		WalletDefaultAddress  func(context.Context) (address.Address, error)
	}
	close func()

	hostname string
}

func (a *api) ClientDealPieceCID(ctx context.Context, root cid.Cid) (DataCIDSize, error) {
	ctx, span := a.span(ctx, "ClientDealPieceCID")
	defer span.End()
	return telemetry.RecordErrorOnSpanTwo[DataCIDSize](span)(a.internal.ClientDealPieceCID(ctx, root))
}

func (a *api) ClientExport(ctx context.Context, exportRef ExportRef, fileRef FileRef) error {
	ctx, span := a.span(ctx, "ClientExport")
	defer span.End()
	return telemetry.RecordErrorOnSpan(span)(a.internal.ClientExport(ctx, exportRef, fileRef))
}

func (a *api) ClientGetDealUpdates(ctx context.Context) (<-chan DealInfo, error) {
	ctx, span := a.span(ctx, "ClientGetDealUpdates")
	defer span.End()
	return telemetry.RecordErrorOnSpanOneChannel[DealInfo](span)(a.internal.ClientGetDealUpdates(ctx))
}

func (a *api) ClientListImports(ctx context.Context) ([]Import, error) {
	ctx, span := a.span(ctx, "ClientListImports")
	defer span.End()
	return telemetry.RecordErrorOnSpanTwo[[]Import](span)(a.internal.ClientListImports(ctx))
}

func (a *api) ClientImport(ctx context.Context, ref FileRef) (*ImportRes, error) {
	ctx, span := a.span(ctx, "ClientImport")
	defer span.End()
	return telemetry.RecordErrorOnSpanTwo[*ImportRes](span)(a.internal.ClientImport(ctx, ref))
}

func (a *api) ClientQueryAsk(ctx context.Context, p peer.ID, miner address.Address) (*StorageAsk, error) {
	ctx, span := a.span(ctx, "ClientQueryAsk")
	defer span.End()
	return telemetry.RecordErrorOnSpanTwo[*StorageAsk](span)(a.internal.ClientQueryAsk(ctx, p, miner))
}

func (a *api) ClientStartDeal(ctx context.Context, params *StartDealParams) (*cid.Cid, error) {
	ctx, span := a.span(ctx, "ClientStartDeal")
	defer span.End()
	return telemetry.RecordErrorOnSpanTwo[*cid.Cid](span)(a.internal.ClientStartDeal(ctx, params))
}

func (a *api) StateGetNetworkParams(ctx context.Context) (*NetworkParams, error) {
	ctx, span := a.span(ctx, "StateGetNetworkParams")
	defer span.End()
	return telemetry.RecordErrorOnSpanTwo[*NetworkParams](span)(a.internal.StateGetNetworkParams(ctx))
}

func (a *api) StateListMiners(ctx context.Context, key TipSetKey) ([]address.Address, error) {
	ctx, span := a.span(ctx, "StateListMiners")
	defer span.End()
	return telemetry.RecordErrorOnSpanTwo[[]address.Address](span)(a.internal.StateListMiners(ctx, key))
}

func (a *api) StateMinerInfo(ctx context.Context, a2 address.Address, key TipSetKey) (MinerInfo, error) {
	ctx, span := a.span(ctx, "StateMinerInfo")
	defer span.End()
	return telemetry.RecordErrorOnSpanTwo[MinerInfo](span)(a.internal.StateMinerInfo(ctx, a2, key))
}

func (a *api) StateMinerPower(ctx context.Context, a2 address.Address, key TipSetKey) (*MinerPower, error) {
	ctx, span := a.span(ctx, "StateMinerPower")
	defer span.End()
	return telemetry.RecordErrorOnSpanTwo[*MinerPower](span)(a.internal.StateMinerPower(ctx, a2, key))
}

func (a *api) Version(ctx context.Context) (APIVersion, error) {
	ctx, span := a.span(ctx, "Version")
	defer span.End()
	return telemetry.RecordErrorOnSpanTwo[APIVersion](span)(a.internal.Version(ctx))
}

func (a *api) WalletDefaultAddress(ctx context.Context) (address.Address, error) {
	ctx, span := a.span(ctx, "WalletDefaultAddress")
	defer span.End()

	return telemetry.RecordErrorOnSpanTwo[address.Address](span)(a.internal.WalletDefaultAddress(ctx))
}

func (a *api) Close() error {
	a.close()
	return nil
}

func (a *api) span(ctx context.Context, method string) (context.Context, trace.Span) {
	return system.NewSpan(
		ctx,
		system.GetTracer(),
		fmt.Sprintf("pkg/publisher/filecoin_lotus/api.api.%s", method),
		trace.WithAttributes(semconv.HostName(a.hostname), semconv.PeerService("lotus")),
		trace.WithSpanKind(trace.SpanKindClient),
	)
}

type TipSetKey struct {
	value string
}

func (k TipSetKey) MarshalJSON() ([]byte, error) {
	return model.JSONMarshalWithMax(k.CIDs())
}

func (k TipSetKey) CIDs() []cid.Cid {
	cids, err := decodeKey([]byte(k.value))
	if err != nil {
		panic("invalid tipset key: " + err.Error())
	}
	return cids
}

// The length of a block header CID in bytes.
var blockHeaderCIDLen int

func init() {
	// hash a large string of zeros so we don't estimate based on inlined CIDs.
	var buf [256]byte
	c, err := abi2.CidBuilder.Sum(buf[:])
	if err != nil {
		panic(err)
	}
	blockHeaderCIDLen = len(c.Bytes())
}

func decodeKey(encoded []byte) ([]cid.Cid, error) {
	// To avoid reallocation of the underlying array, estimate the number of CIDs to be extracted
	// by dividing the encoded length by the expected CID length.
	estimatedCount := len(encoded) / blockHeaderCIDLen
	cids := make([]cid.Cid, 0, estimatedCount)
	nextIdx := 0
	for nextIdx < len(encoded) {
		nr, c, err := cid.CidFromBytes(encoded[nextIdx:])
		if err != nil {
			return nil, err
		}
		cids = append(cids, c)
		nextIdx += nr
	}
	return cids, nil
}

type MinerPower struct {
	MinerPower  Claim
	TotalPower  Claim
	HasMinPower bool
}

type Claim struct {
	// Sum of raw byte power for a miner's sectors.
	RawBytePower abi2.StoragePower

	// Sum of quality adjusted power for a miner's sectors.
	QualityAdjPower abi2.StoragePower
}

type FileRef struct {
	Path  string
	IsCAR bool
}

type ImportRes struct {
	Root     cid.Cid
	ImportID ID
}

type ID uint64

type StartDealParams struct {
	Data               *DataRef
	Wallet             address.Address
	Miner              address.Address
	EpochPrice         big2.Int
	MinBlocksDuration  uint64
	ProviderCollateral big2.Int
	DealStartEpoch     ChainEpoch
	FastRetrieval      bool
	VerifiedDeal       bool
}

type ChainEpoch int64

type DataRef struct {
	TransferType string
	Root         cid.Cid

	PieceCid     *cid.Cid               // Optional for non-manual transfer, will be recomputed from the data if not given
	PieceSize    abi2.UnpaddedPieceSize // Optional for non-manual transfer, will be recomputed from the data if not given
	RawBlockSize uint64                 // Optional: used as the denominator when calculating transfer %
}

type DealInfo struct {
	ProposalCid cid.Cid
	State       storagemarket.StorageDealStatus
	Message     string // more information about deal state, particularly errors
	Provider    address.Address

	DataRef  *storagemarket.DataRef
	PieceCID cid.Cid
	Size     uint64

	PricePerEpoch big2.Int
	Duration      uint64

	DealID abi2.DealID

	CreationTime time.Time
	Verified     bool
}

type FIL big2.Int

type APIVersion struct {
	Version    string
	APIVersion Version
	BlockDelay uint64
}

type Version uint32

type DataCIDSize struct {
	PayloadSize int64
	PieceSize   abi2.PaddedPieceSize
	PieceCID    cid.Cid
}

type StorageAsk struct {
	Response *storagemarket.StorageAsk

	DealProtocols []string
}

type MinerInfo struct {
	Owner                      address.Address   // Must be an ID-address.
	Worker                     address.Address   // Must be an ID-address.
	NewWorker                  address.Address   // Must be an ID-address.
	ControlAddresses           []address.Address // Must be an ID-addresses.
	WorkerChangeEpoch          int64
	PeerId                     *peer.ID
	Multiaddrs                 [][]byte
	WindowPoStProofType        int64
	SectorSize                 uint64
	WindowPoStPartitionSectors uint64
	ConsensusFaultElapsed      int64
	Beneficiary                address.Address
	BeneficiaryTerm            *miner.BeneficiaryTerm
	PendingBeneficiaryTerm     *miner.PendingBeneficiaryChange
}

type NetworkParams struct {
	NetworkName    string
	BlockDelaySecs uint64
}

type Import struct {
	Key      ID
	Err      string
	Root     *cid.Cid
	Source   string
	FilePath string
	CARPath  string
}

type Selector string

type DagSpec struct {
	DataSelector      *Selector
	ExportMerkleProof bool
}

type ExportRef struct {
	Root         cid.Cid
	DAGs         []DagSpec
	FromLocalCAR string
	DealID       retrievalmarket.DealID
}
