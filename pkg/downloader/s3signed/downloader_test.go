//go:build integration || !unit

package s3signed

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/http"
	"github.com/bacalhau-project/bacalhau/pkg/lib/gzip"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	s3test "github.com/bacalhau-project/bacalhau/pkg/s3/test"
)

type DownloaderTestSuite struct {
	*s3test.HelperSuite
	downloader *Downloader
	signer     *s3helper.ResultSigner
}

func (s *DownloaderTestSuite) SetupSuite() {
	s.HelperSuite.SetupSuite()
	s.signer = s3helper.NewResultSigner(s3helper.ResultSignerParams{
		ClientProvider: s.ClientProvider,
		Expiration:     5 * time.Minute,
	})
	s.downloader = NewDownloader(DownloaderParams{
		HTTPDownloader: http.NewHTTPDownloader(),
	})
}

func (s *DownloaderTestSuite) TearDownSuite() {
	s.Destroy()
}

func TestDownloaderTestSuite(t *testing.T) {
	helperSuite := s3test.NewTestHelper(t, s3test.HelperSuiteParams{
		BasePrefix: "integration-tests-s3downloader",
	})
	suite.Run(t, &DownloaderTestSuite{HelperSuite: helperSuite})
}

func (s *DownloaderTestSuite) TestIsInstalled() {
	ctx := context.Background()
	res, err := s.downloader.IsInstalled(ctx)
	s.Require().NoError(err)
	s.True(res)
}

func (s *DownloaderTestSuite) TestDownloadCompressed() {
	storageSpec, resultPath := s.PrepareAndPublish(s3helper.EncodingGzip)
	s.T().Log(resultPath)

	// get pre-signed url
	s.Require().NoError(s.signer.Transform(s.Ctx, &storageSpec))
	s.Require().Equal(models.StorageSourceS3PreSigned, storageSpec.Type)

	// download signed url
	downloadParentPath, err := os.MkdirTemp(s.TempDir, "")
	s.Require().NoError(err)

	downloadedFile, err := s.downloader.FetchResult(s.Ctx, downloader.DownloadItem{
		Result:     &storageSpec,
		ParentPath: downloadParentPath,
	})
	s.Require().NoError(err)

	// compare downloaded file with original
	decompressedPath, err := os.MkdirTemp(s.TempDir, "")
	s.Require().NoError(err)
	s.Require().NoError(gzip.Decompress(downloadedFile, decompressedPath))

	s3test.AssertEqualDirectories(s.T(), resultPath, decompressedPath)
}

func (s *DownloaderTestSuite) TestDownloadUncompressed() {
	storageSpec, resultPath := s.PrepareAndPublish(s3helper.EncodingPlain)
	s.T().Log(resultPath)

	// get a non-signed url
	s.Require().NoError(s.signer.Transform(s.Ctx, &storageSpec))
	s.Require().Equal(models.StorageSourceS3, storageSpec.Type)

	// download signed url
	downloadParentPath, err := os.MkdirTemp(s.TempDir, "")
	s.Require().NoError(err)

	_, err = s.downloader.FetchResult(s.Ctx, downloader.DownloadItem{
		Result:     &storageSpec,
		ParentPath: downloadParentPath,
	})
	s.Require().Error(err)
}
