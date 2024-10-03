//go:build integration || !unit

package ipfs

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

// how many bytes more does ipfs report the file than the actual content?
const IpfsMetadataSize uint64 = 8

type StorageSuite struct {
	suite.Suite
	ipfsClient *ipfs.Client
	storage    *StorageProvider
}

func TestStorageSuite(t *testing.T) {
	suite.Run(t, new(StorageSuite))
}

func (s *StorageSuite) SetupSuite() {
	logger.ConfigureTestLogging(s.T())

	endpoint := testutils.MustHaveIPFSEndpoint(s.T())

	var err error
	s.ipfsClient, err = ipfs.NewClient(context.Background(), endpoint)
	s.storage, err = NewStorage(*s.ipfsClient, 5*time.Second)
	s.Require().NoError(err)
}

func (s *StorageSuite) TestGetVolumeSize() {
	ctx := context.Background()
	for _, testString := range []string{
		"hello from test volume size",
		"hello world",
	} {
		s.Run(testString, func() {
			cid, err := ipfs.AddTextToNodes(ctx, []byte(testString), *s.ipfsClient)
			s.Require().NoError(err)

			result, err := s.storage.GetVolumeSize(ctx, models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceIPFS,
					Params: Source{
						CID: cid,
					}.ToMap(),
				},
				Target: "/",
			})

			s.Require().NoError(err)
			s.Require().Equal(uint64(len(testString))+IpfsMetadataSize, result)
		})
	}
}

func (s *StorageSuite) TestPrepareStorageRespectsTimeouts() {
	for _, testDuration := range []time.Duration{
		// 0, // Disable test -- timeouts aren't respected when getting cached files
		time.Minute,
	} {
		s.Run(fmt.Sprint(testDuration), func() {
			ctx, cancel := context.WithTimeout(context.Background(), testDuration)
			defer cancel()

			cid, err := ipfs.AddTextToNodes(ctx, []byte("testString"), *s.ipfsClient)
			s.Require().NoError(err)

			_, err = s.storage.PrepareStorage(ctx, s.T().TempDir(), models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceIPFS,
					Params: Source{
						CID: cid,
					}.ToMap(),
				},
				Target: "/",
			})
			s.Require().Equal(testDuration == 0, err != nil)
		})
	}
}

func (s *StorageSuite) TestGetVolumeSizeRespectsTimeout() {
	for _, testDuration := range []time.Duration{
		// 0, // Disable test -- timeouts aren't respected when getting cached files
		time.Minute,
	} {
		s.Run(fmt.Sprint(testDuration), func() {
			ctx := context.Background()

			cid, err := ipfs.AddTextToNodes(ctx, []byte("testString"), *s.ipfsClient)
			s.Require().NoError(err)

			_, err = s.storage.GetVolumeSize(ctx, models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceIPFS,
					Params: Source{
						CID: cid,
					}.ToMap(),
				},
				Target: "/",
			})

			s.Require().Equal(testDuration == 0, err != nil)
		})
	}
}
