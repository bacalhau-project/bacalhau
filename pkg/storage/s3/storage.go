package s3

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
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

type StorageProviderParams struct {
	ClientProvider *s3helper.ClientProvider
}

type StorageProvider struct {
	clientProvider *s3helper.ClientProvider
	timeout        time.Duration
}

func NewStorage(getVolumeTimeout time.Duration, provider *s3helper.ClientProvider) *StorageProvider {
	return &StorageProvider{
		clientProvider: provider,
		timeout:        getVolumeTimeout,
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
func (s *StorageProvider) HasStorageLocally(_ context.Context, _ models.InputSource) (bool, error) {
	// TODO: return true if the content is on the same AZ or datacenter as the host
	return false, nil
}

func (s *StorageProvider) GetVolumeSize(ctx context.Context, execution *models.Execution, volume models.InputSource) (uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	source, err := s3helper.DecodeSourceSpec(volume.Source)
	if err != nil {
		return 0, err
	}

	client := s.clientProvider.GetClient(source.Endpoint, source.Region)
	objects, err := s.explodeKey(ctx, client, source)
	if err != nil {
		return 0, err
	}

	objects, err = s3helper.PartitionObjects(objects, execution.Job.Count, execution.PartitionIndex, source)
	if err != nil {
		return 0, err
	}

	var size uint64
	for _, object := range objects {
		// Check for negative size
		if object.Size < 0 {
			return 0, fmt.Errorf("invalid negative size for object: %d", object.Size)
		}

		// Check for overflow
		// MaxUint64 - size = remaining space before overflow

		if object.Size > 0 && uint64(object.Size) > math.MaxUint64-size {
			return 0, fmt.Errorf("total size exceeds uint64 maximum")
		}

		size += uint64(object.Size)
	}
	return size, nil
}

func (s *StorageProvider) PrepareStorage(
	ctx context.Context,
	storageDirectory string,
	execution *models.Execution,
	input models.InputSource) (storage.StorageVolume, error) {
	source, err := s3helper.DecodeSourceSpec(input.Source)
	if err != nil {
		return storage.StorageVolume{}, err
	}
	log.Debug().Msgf("Preparing storage for s3://%s/%s", source.Bucket, source.Key)

	// create random directory within the provided directory to store the content
	// and to avoid conflicts with other downloads. If we wanted all downloads from
	// s3 to be allowed just in `storagePath` we'd have to be sure the names didn't
	// clash.
	outputDir, err := os.MkdirTemp(storageDirectory, "s3-input-*")
	if err != nil {
		return storage.StorageVolume{}, err
	}

	client := s.clientProvider.GetClient(source.Endpoint, source.Region)
	objects, err := s.explodeKey(ctx, client, source)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	objects, err = s3helper.PartitionObjects(objects, execution.Job.Count, execution.PartitionIndex, source)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	prefixTokens := strings.Split(s.sanitizeKey(source.Key), "/")

	for _, object := range objects {
		err = s.downloadObject(ctx, client, source, object, outputDir, prefixTokens)
		if err != nil {
			return storage.StorageVolume{}, err
		}
	}

	volume := storage.StorageVolume{
		Type:   storage.StorageVolumeConnectorBind,
		Source: outputDir,
		Target: input.Target,
	}

	return volume, nil
}

// downloadObject downloads a single object from S3 to local disk
func (s *StorageProvider) downloadObject(ctx context.Context,
	client *s3helper.ClientWrapper,
	source s3helper.SourceSpec,
	object s3helper.ObjectSummary,
	parentDir string,
	prefixTokens []string) error {
	// trim the user supplied prefix from the object local path
	objectTokens := strings.Split(*object.Key, "/")
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

	// create all parent directories if needed
	err := os.MkdirAll(filepath.Dir(outputPath), models.DownloadFolderPerm)
	if err != nil {
		return err
	}

	// create the file to download to
	outputFile, err := os.OpenFile(outputPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, models.DownloadFilePerm) //nolint:gosec // G304: Caller responsible for validating output path
	if err != nil {
		return err
	}
	defer func() { _ = outputFile.Close() }() //nolint:errcheck

	log.Debug().Msgf("Downloading s3://%s/%s versionID:%s, eTag:%s to %s.",
		source.Bucket, aws.ToString(object.Key), aws.ToString(object.VersionID), aws.ToString(object.ETag), outputFile.Name())
	_, err = client.Downloader.Download(ctx, outputFile, &s3.GetObjectInput{
		Bucket:    aws.String(source.Bucket),
		Key:       object.Key,
		VersionId: object.VersionID,
		IfMatch:   object.ETag,
	})
	if err != nil {
		return s3helper.NewS3InputSourceServiceError(err)
	}
	return nil
}

func (s *StorageProvider) CleanupStorage(_ context.Context, _ models.InputSource, volume storage.StorageVolume) error {
	fileInfo, err := os.Stat(volume.Source)
	if err != nil {
		return err
	}

	if fileInfo.IsDir() {
		return os.RemoveAll(volume.Source)
	}

	return os.Remove(volume.Source)
}

func (s *StorageProvider) Upload(_ context.Context, _ string) (models.SpecConfig, error) {
	return models.SpecConfig{}, fmt.Errorf("not implemented")
}

func (s *StorageProvider) explodeKey(
	ctx context.Context, client *s3helper.ClientWrapper, storageSpec s3helper.SourceSpec) (
	[]s3helper.ObjectSummary, error) {
	if storageSpec.Key != "" && !strings.HasSuffix(storageSpec.Key, "*") && !strings.HasSuffix(storageSpec.Key, "/") {
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
			return nil, s3helper.NewS3InputSourceServiceError(err)
		}

		if storageSpec.ChecksumSHA256 != "" && storageSpec.ChecksumSHA256 != aws.ToString(headResp.ChecksumSHA256) {
			return nil, fmt.Errorf("checksum mismatch for s3://%s/%s, expected %s, got %s",
				storageSpec.Bucket, storageSpec.Key, storageSpec.ChecksumSHA256, aws.ToString(headResp.ChecksumSHA256))
		}
		if headResp.ContentType != nil && !strings.HasPrefix(*headResp.ContentType, "application/x-directory") {
			objectSummary := s3helper.ObjectSummary{
				Key:  aws.String(storageSpec.Key),
				Size: *headResp.ContentLength,
				ETag: headResp.ETag,
			}
			if storageSpec.VersionID != "" {
				objectSummary.VersionID = aws.String(storageSpec.VersionID)
			}
			return []s3helper.ObjectSummary{objectSummary}, nil
		}
	}

	// Compile the regex pattern
	regex, err := regexp.Compile(storageSpec.Filter)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	// if the key is a directory, or ends with a wildcard, we need to list the objects starting with the key
	sanitizedKey := s.sanitizeKey(storageSpec.Key)
	res := make([]s3helper.ObjectSummary, 0)
	var continuationToken *string
	for {
		resp, err := client.S3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(storageSpec.Bucket),
			Prefix:            aws.String(sanitizedKey),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return nil, s3helper.NewS3InputSourceServiceError(err)
		}
		for _, object := range resp.Contents {
			if storageSpec.Filter != "" {
				trimmedKey := strings.TrimPrefix(aws.ToString(object.Key), sanitizedKey)
				if !regex.MatchString(trimmedKey) {
					continue
				}
			}
			res = append(res, s3helper.ObjectSummary{
				Key:   object.Key,
				Size:  *object.Size,
				IsDir: strings.HasSuffix(*object.Key, "/"),
			})
		}
		if !*resp.IsTruncated {
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
