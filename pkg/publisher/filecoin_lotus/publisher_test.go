package filecoinlotus

import (
	"context"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus/api"
	"github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus/api/storagemarket"
	"github.com/filecoin-project/go-address"
	abi2 "github.com/filecoin-project/go-state-types/abi"
	big2 "github.com/filecoin-project/go-state-types/big"
	"github.com/golang/mock/gomock"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type PublisherTestSuite struct {
	suite.Suite
	executor *Publisher
	client   *MockClient
}

// This test aims to ensure the publisher is covered by _some_ tests, as the tests in `pkg/test/devstack/lotus_test.go`
// are flaking in CI so are currently skipped.
func TestPublisherTestSuite(t *testing.T) {
	suite.Run(t, new(PublisherTestSuite))
}

func (s *PublisherTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.T().Cleanup(ctrl.Finish)
	s.client = NewMockClient(ctrl)
	s.executor = newPublisher(PublisherConfig{
		StorageDuration: 1 * time.Hour,
		MaximumPing:     1 * time.Second,
	}, s.client)
}

func (s *PublisherTestSuite) TestIsInstalled() {
	s.client.EXPECT().Version(gomock.Any()).Return(api.APIVersion{Version: "hello"}, nil)
	actual, err := s.executor.IsInstalled(context.Background())
	s.NoError(err)
	s.True(actual)
}

func (s *PublisherTestSuite) TestPublishToLotus() {
	contentCid := cid.MustParse("bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi")
	add, err := address.NewIDAddress(1234)
	s.Require().NoError(err)

	gomock.InOrder(
		s.client.EXPECT().
			ClientImport(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, ref api.FileRef) (*api.ImportRes, error) {
				s.True(ref.IsCAR)
				return &api.ImportRes{
					Root:     contentCid,
					ImportID: 0,
				}, nil
			}),
		s.client.EXPECT().ClientDealPieceCID(gomock.Any(), contentCid).Return(api.DataCIDSize{
			PieceSize: 512,
			PieceCID:  contentCid,
		}, nil),
		s.client.EXPECT().StateGetNetworkParams(gomock.Any()).Return(&api.NetworkParams{BlockDelaySecs: 30}, nil),
		s.client.EXPECT().WalletDefaultAddress(gomock.Any()).Return(add, nil),
		s.client.EXPECT().StateListMiners(gomock.Any(), gomock.Any()).Return([]address.Address{add}, nil),
		s.client.EXPECT().StateMinerInfo(gomock.Any(), add, gomock.Any()).Return(api.MinerInfo{
			PeerId: pointer[peer.ID]("4321"),
		}, nil),
		s.client.EXPECT().StateMinerPower(gomock.Any(), add, gomock.Any()).Return(&api.MinerPower{HasMinPower: true}, nil),
		s.client.EXPECT().
			ClientQueryAsk(gomock.Any(), peer.ID("4321"), add).
			Return(&api.StorageAsk{Response: &storagemarket.StorageAsk{
				Price:        big2.NewInt(512),
				MinPieceSize: 128,
				MaxPieceSize: 1024,
			}}, nil),
		s.client.EXPECT().
			ClientStartDeal(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, deal *api.StartDealParams) (*cid.Cid, error) {
				s.Equal("graphsync", deal.Data.TransferType)
				s.Equal(contentCid, deal.Data.Root)
				s.Equal(contentCid, *deal.Data.PieceCid)
				s.Equal(abi2.UnpaddedPieceSize(508), deal.Data.PieceSize)
				s.Equal(add, deal.Wallet)
				s.Equal(add, deal.Miner)
				s.Equal(big2.NewInt(0), deal.EpochPrice)
				s.Equal(uint64(120), deal.MinBlocksDuration)
				return &contentCid, nil
			}),
		s.client.EXPECT().ClientGetDealUpdates(gomock.Any()).DoAndReturn(func(context.Context) (<-chan api.DealInfo, error) {
			c := make(chan api.DealInfo, 2)
			c <- api.DealInfo{
				ProposalCid: contentCid,
				State:       storagemarket.StorageDealAcceptWait,
			}
			c <- api.DealInfo{
				ProposalCid: contentCid,
				State:       storagemarket.StorageDealCheckForAcceptance,
			}
			return c, nil
		}),
	)

	resultsDir := s.T().TempDir()
	s.Require().NoError(os.WriteFile(filepath.Join(resultsDir, "hello.txt"), []byte("world"), 0644))

	spec, err := s.executor.PublishShardResult(context.Background(), model.JobShard{
		Job:   &model.Job{Metadata: model.Metadata{ID: "foo"}},
		Index: 0,
	}, "1234", resultsDir)
	s.Require().NoError(err)

	s.Equal(contentCid.String(), spec.CID)
}

func pointer[T any](t T) *T {
	return &t
}

//go:generate go run github.com/golang/mock/mockgen -destination mock_test.go -package filecoinlotus -write_package_comment=false -source ./api/api.go Client
