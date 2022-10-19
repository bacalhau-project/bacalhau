//nolint:gochecknoinits,stylecheck // Most of the code in this package is copied with hope to use the upstream version
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus/api/storagemarket"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	abi2 "github.com/filecoin-project/go-state-types/abi"
	big2 "github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/builtin/v9/miner"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
)

// Importing the Lotus API causes dependency conflicts between this project and Lotus
// So 're-implement' the API here to avoid the conflicts
// https://github.com/filecoin-project/lotus/blob/master/api/api_full.go

func NewClient(ctx context.Context, host string, token string) (Client, error) {
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

	return &client, nil
}

type Client interface {
	ClientDealPieceCID(ctx context.Context, root cid.Cid) (DataCIDSize, error)
	ClientGetDealInfo(context.Context, cid.Cid) (*DealInfo, error)
	ClientImport(context.Context, FileRef) (*ImportRes, error)
	ClientQueryAsk(ctx context.Context, p peer.ID, miner address.Address) (*StorageAsk, error)
	ClientStartDeal(context.Context, *StartDealParams) (*cid.Cid, error)
	StateGetNetworkParams(ctx context.Context) (*NetworkParams, error)
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
		ClientGetDealInfo     func(context.Context, cid.Cid) (*DealInfo, error)
		ClientImport          func(context.Context, FileRef) (*ImportRes, error)
		ClientQueryAsk        func(context.Context, peer.ID, address.Address) (*StorageAsk, error)
		ClientStartDeal       func(context.Context, *StartDealParams) (*cid.Cid, error)
		StateGetNetworkParams func(context.Context) (*NetworkParams, error)
		StateMinerInfo        func(context.Context, address.Address, TipSetKey) (MinerInfo, error)
		StateMinerPower       func(context.Context, address.Address, TipSetKey) (*MinerPower, error)
		Version               func(context.Context) (APIVersion, error)
		WalletDefaultAddress  func(context.Context) (address.Address, error)
	}
	close func()
}

func (a *api) ClientDealPieceCID(ctx context.Context, root cid.Cid) (DataCIDSize, error) {
	return a.internal.ClientDealPieceCID(ctx, root)
}

func (a *api) ClientGetDealInfo(ctx context.Context, cid cid.Cid) (*DealInfo, error) {
	return a.internal.ClientGetDealInfo(ctx, cid)
}

func (a *api) ClientImport(ctx context.Context, ref FileRef) (*ImportRes, error) {
	return a.internal.ClientImport(ctx, ref)
}

func (a *api) ClientQueryAsk(ctx context.Context, p peer.ID, miner address.Address) (*StorageAsk, error) {
	return a.internal.ClientQueryAsk(ctx, p, miner)
}

func (a *api) ClientStartDeal(ctx context.Context, params *StartDealParams) (*cid.Cid, error) {
	return a.internal.ClientStartDeal(ctx, params)
}

func (a *api) StateGetNetworkParams(ctx context.Context) (*NetworkParams, error) {
	return a.internal.StateGetNetworkParams(ctx)
}

func (a *api) StateMinerInfo(ctx context.Context, a2 address.Address, key TipSetKey) (MinerInfo, error) {
	return a.internal.StateMinerInfo(ctx, a2, key)
}

func (a *api) StateMinerPower(ctx context.Context, a2 address.Address, key TipSetKey) (*MinerPower, error) {
	return a.internal.StateMinerPower(ctx, a2, key)
}

func (a *api) Version(ctx context.Context) (APIVersion, error) {
	return a.internal.Version(ctx)
}

func (a *api) WalletDefaultAddress(ctx context.Context) (address.Address, error) {
	return a.internal.WalletDefaultAddress(ctx)
}

func (a *api) Close() error {
	a.close()
	return nil
}

type TipSetKey struct {
	value string
}

func (k TipSetKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.Cids())
}

func (k TipSetKey) Cids() []cid.Cid {
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
