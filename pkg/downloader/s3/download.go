package s3

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	s32 "github.com/bacalhau-project/bacalhau/pkg/s3"
)

type Downloader struct {
	provider *s32.ClientProvider
}

func (d *Downloader) IsInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

func NewDownloader(provider *s32.ClientProvider) *Downloader {
	return &Downloader{provider: provider}
}

// TODO this would be WAY easier if we TAR'd the file before uploading
func (d *Downloader) FetchResult(ctx context.Context, item downloader.DownloadItem) (string, error) {
	resultSpec, err := s32.DecodeSourceSpec(item.Result)
	if err != nil {
		return "", err
	}

	client := d.provider.GetClient(resultSpec.Endpoint, resultSpec.Region)

	// List all objects in a bucket
	paginator := s3.NewListObjectsV2Paginator(client.S3, &s3.ListObjectsV2Input{Bucket: &resultSpec.Bucket})
	dwnld := manager.NewDownloader(client.S3)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get page: %w", err)
		}

		for _, obj := range page.Contents {
			downloadPath := filepath.Join(item.ParentPath, *obj.Key) // local directory path
			if err := d.downloadFile(ctx, dwnld, resultSpec.Bucket, *obj.Key, downloadPath); err != nil {
				return "", fmt.Errorf("failed to download file, %w", err)
			}
		}
	}

	return item.ParentPath, nil
}

func (d *Downloader) downloadFile(ctx context.Context, dwnld *manager.Downloader, bucket, key, downloadPath string) error {
	// Create the directories in the path
	if err := os.MkdirAll(filepath.Dir(downloadPath), os.ModePerm); err != nil {
		return err
	}

	// Create a file to write the S3 Object contents to.
	f, err := os.Create(downloadPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write the contents of S3 Object to the file
	if _, err := dwnld.Download(ctx, f, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}); err != nil {
		return err
	}

	return err
}
