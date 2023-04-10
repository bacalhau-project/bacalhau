package s3

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/rs/zerolog/log"
)

/*
Storage provider that supports fetching content from an S3 compatible storage provider.
Since users can define endpoint to download from, the storage provide supports downloading from S3 service,
or from MinIO, Ceph, SeaweedFS, etc.

The storage provider supports downloading:
- a single object: s3://myBucket/dir/file-001.txt
- a directory and all its content: s3://myBucket/dir/
- a prefix and all objects matching the prefix: s3://myBucket/dir/file-*
*/

type s3ObjectSummary struct {
	key       *string
	eTag      *string
	versionID *string
	size      int64
	isDir     bool
}

type StorageProviderParams struct {
	LocalDir       string
	ClientProvider *s3helper.ClientProvider
}

type StorageProvider struct {
	localDir       string
	clientProvider *s3helper.ClientProvider
}

func NewStorage(params StorageProviderParams) *StorageProvider {
	return &StorageProvider{
		localDir:       params.LocalDir,
		clientProvider: params.ClientProvider,
	}
}

// IsInstalled checks if the storage provider is installed
// We assume that the storage provider is installed if the host has AWS credentials configured, which includes:
// - Configuring the AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables
// - Configuring credentials in ~/.aws/credentials
// - Configuring credentials in the EC2 instance metadata service, assuming the host is running on EC2
func (s *StorageProvider) IsInstalled(_ context.Context) (bool, error) {
	return s.clientProvider.IsInstalled(), nil
}

// HasStorageLocally checks if the requested content is hosted locally.
func (s *StorageProvider) HasStorageLocally(_ context.Context, _ model.StorageSpec) (bool, error) {
	// TODO: return true if the content is on the same AZ or datacenter as the host
	return false, nil
}

func (s *StorageProvider) GetVolumeSize(ctx context.Context, volume model.StorageSpec) (uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, config.GetVolumeSizeRequestTimeout(ctx))
	defer cancel()

	client := s.clientProvider.GetClient(volume.S3.Endpoint, volume.S3.Region)
	objects, err := s.explodeKey(ctx, client, volume.S3)
	if err != nil {
		return 0, err
	}
	size := uint64(0)
	for _, object := range objects {
		size += uint64(object.size)
	}
	return size, nil
}

func (s *StorageProvider) PrepareStorage(ctx context.Context, storageSpec model.StorageSpec) (storage.StorageVolume, error) {
	log.Debug().Msgf("Preparing storage for s3://%s/%s", storageSpec.S3.Bucket, storageSpec.S3.Key)

	// create random directory to store the content and to avoid conflicts with other downloads
	outputDir, err := os.MkdirTemp(s.localDir, "s3-input-*")
	if err != nil {
		return storage.StorageVolume{}, err
	}

	client := s.clientProvider.GetClient(storageSpec.S3.Endpoint, storageSpec.S3.Region)
	objects, err := s.explodeKey(ctx, client, storageSpec.S3)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	prefixTokens := strings.Split(s.sanitizeKey(storageSpec.S3.Key), "/")

	for _, object := range objects {
		err = s.downloadObject(ctx, client, storageSpec, object, outputDir, prefixTokens)
		if err != nil {
			return storage.StorageVolume{}, err
		}
	}

	volume := storage.StorageVolume{
		Type:   storage.StorageVolumeConnectorBind,
		Source: outputDir,
		Target: storageSpec.Path,
	}

	return volume, nil
}

// downloadObject downloads a single object from S3 to local disk
func (s *StorageProvider) downloadObject(ctx context.Context,
	client *s3helper.ClientWrapper,
	storageSpec model.StorageSpec,
	object s3ObjectSummary,
	parentDir string,
	prefixTokens []string) error {
	// trim the user supplied prefix from the object local path
	objectTokens := strings.Split(*object.key, "/")
	startingIndex := 0
	for i := 0; i < len(prefixTokens)-1; i++ {
		if prefixTokens[i] == objectTokens[i] {
			startingIndex++
		} else {
			break
		}
	}

	// relative output path to the supplied prefix
	outputPath := filepath.Join(parentDir, filepath.Join(objectTokens[startingIndex:]...))

	if object.isDir {
		return os.MkdirAll(outputPath, model.DownloadFolderPerm)
	}

	// create all parent directories if needed
	err := os.MkdirAll(filepath.Dir(outputPath), model.DownloadFolderPerm)
	if err != nil {
		return err
	}

	// create the file to download to
	outputFile, err := os.OpenFile(outputPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, model.DownloadFilePerm)
	if err != nil {
		return err
	}
	defer outputFile.Close() //nolint:errcheck

	log.Debug().Msgf("Downloading s3://%s/%s versionID:%s, eTag:%s to %s.",
		storageSpec.S3.Bucket, aws.ToString(object.key), aws.ToString(object.versionID), aws.ToString(object.eTag), outputFile.Name())
	_, err = client.Downloader.Download(ctx, outputFile, &s3.GetObjectInput{
		Bucket:    aws.String(storageSpec.S3.Bucket),
		Key:       object.key,
		VersionId: object.versionID,
		IfMatch:   object.eTag,
	})
	return err
}

func (s *StorageProvider) CleanupStorage(_ context.Context, _ model.StorageSpec, volume storage.StorageVolume) error {
	return os.RemoveAll(volume.Source)
}

func (s *StorageProvider) Upload(_ context.Context, _ string) (model.StorageSpec, error) {
	return model.StorageSpec{}, fmt.Errorf("not implemented")
}

func (s *StorageProvider) explodeKey(
	ctx context.Context, client *s3helper.ClientWrapper, storageSpec *model.S3StorageSpec) ([]s3ObjectSummary, error) {
	if !strings.HasSuffix(storageSpec.Key, "*") {
		request := &s3.HeadObjectInput{
			Bucket: aws.String(storageSpec.Bucket),
			Key:    aws.String(storageSpec.Key),
		}
		if storageSpec.VersionID != "" {
			request.VersionId = aws.String(storageSpec.VersionID)
		}
		if storageSpec.ChecksumSHA256 != "" {
			request.ChecksumMode = types.ChecksumModeEnabled
		}

		headResp, err := client.S3.HeadObject(ctx, request)
		if err != nil {
			return nil, err
		}

		if storageSpec.ChecksumSHA256 != "" && storageSpec.ChecksumSHA256 != aws.ToString(headResp.ChecksumSHA256) {
			return nil, fmt.Errorf("checksum mismatch for s3://%s/%s, expected %s, got %s",
				storageSpec.Bucket, storageSpec.Key, storageSpec.ChecksumSHA256, aws.ToString(headResp.ChecksumSHA256))
		}
		if headResp.ContentType != nil && !strings.HasPrefix(*headResp.ContentType, "application/x-directory") {
			objectSummary := s3ObjectSummary{
				key:  aws.String(storageSpec.Key),
				size: headResp.ContentLength,
				eTag: headResp.ETag,
			}
			if storageSpec.VersionID != "" {
				objectSummary.versionID = aws.String(storageSpec.VersionID)
			}
			return []s3ObjectSummary{objectSummary}, nil
		}
	}

	// if the key is a directory, or ends with a wildcard, we need to list the objects starting with the key
	sanitizedKey := s.sanitizeKey(storageSpec.Key)
	res := make([]s3ObjectSummary, 0)
	var continuationToken *string
	for {
		resp, err := client.S3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(storageSpec.Bucket),
			Prefix:            aws.String(sanitizedKey),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return nil, err
		}
		for _, object := range resp.Contents {
			res = append(res, s3ObjectSummary{
				key:   object.Key,
				size:  object.Size,
				isDir: strings.HasSuffix(*object.Key, "/"),
			})
		}
		if !resp.IsTruncated {
			break
		}
		continuationToken = resp.NextContinuationToken
	}
	return res, nil
}

func (s *StorageProvider) sanitizeKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.TrimSuffix(key, "*")
	return key
}

// Compile time interface check:
var _ storage.Storage = (*StorageProvider)(nil)
