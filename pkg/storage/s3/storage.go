package s3

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	sync "github.com/bacalhau-project/golang-mutex-tracer"
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

type s3ClientWrapper struct {
	s3         *s3.Client
	downloader *manager.Downloader
	endpoint   string
	region     string
}
type s3ObjectSummary struct {
	key   *string
	size  int64
	isDir bool
}

type StorageProviderParams struct {
	LocalDir  string
	AWSConfig aws.Config
}

type StorageProvider struct {
	localDir  string
	awsConfig aws.Config
	clients   map[string]*s3ClientWrapper
	mu        sync.RWMutex
}

func NewStorage(params StorageProviderParams) *StorageProvider {
	s := &StorageProvider{
		localDir:  params.LocalDir,
		awsConfig: params.AWSConfig,
		clients:   make(map[string]*s3ClientWrapper),
	}
	s.mu.EnableTracerWithOpts(sync.Opts{
		Threshold: 50 * time.Millisecond,
		Id:        "S3StorageProvider.mu",
	})
	return s
}

// IsInstalled checks if the storage provider is installed
// We assume that the storage provider is installed if the host has AWS credentials configured, which includes:
// - Configuring the AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables
// - Configuring credentials in ~/.aws/credentials
// - Configuring credentials in the EC2 instance metadata service, assuming the host is running on EC2
func (s *StorageProvider) IsInstalled(_ context.Context) (bool, error) {
	return HasValidCredentials(s.awsConfig), nil
}

// HasStorageLocally checks if the requested content is hosted locally.
func (s *StorageProvider) HasStorageLocally(_ context.Context, _ model.StorageSpec) (bool, error) {
	// TODO: return true if the content is on the same AZ or datacenter as the host
	return false, nil
}

func (s *StorageProvider) GetVolumeSize(ctx context.Context, volume model.StorageSpec) (uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, config.GetVolumeSizeRequestTimeout(ctx))
	defer cancel()

	client := s.getClient(volume)
	objects, err := s.explodeKey(ctx, client, volume.S3.Bucket, volume.S3.Key)
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

	client := s.getClient(storageSpec)
	objects, err := s.explodeKey(ctx, client, storageSpec.S3.Bucket, storageSpec.S3.Key)
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
	client *s3ClientWrapper,
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

	log.Debug().Msgf("Downloading %s to %s", *object.key, outputFile.Name())
	_, err = client.downloader.Download(ctx, outputFile, &s3.GetObjectInput{
		Bucket: aws.String(storageSpec.S3.Bucket),
		Key:    object.key,
	})
	return err
}

func (s *StorageProvider) CleanupStorage(_ context.Context, _ model.StorageSpec, volume storage.StorageVolume) error {
	return os.RemoveAll(volume.Source)
}

func (s *StorageProvider) Upload(_ context.Context, _ string) (model.StorageSpec, error) {
	return model.StorageSpec{}, fmt.Errorf("not implemented")
}

// getClient creates and cached client for the given storage spec endpoint and region
func (s *StorageProvider) getClient(storageSpec model.StorageSpec) *s3ClientWrapper {
	clientIdentifier := fmt.Sprintf("%s-%s", storageSpec.S3.Endpoint, storageSpec.S3.Region)
	s.mu.RLock()
	client, ok := s.clients[clientIdentifier]
	s.mu.RUnlock()
	if ok {
		return client
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check again in case another goroutine created the client while we were waiting for the lock.
	client, ok = s.clients[clientIdentifier]
	if ok {
		return client
	}

	s3Config := s.awsConfig.Copy()
	if storageSpec.S3.Region != "" {
		s3Config.Region = storageSpec.S3.Region
	}
	if storageSpec.S3.Endpoint != "" {
		s3Config.EndpointResolverWithOptions =
			aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...any) (aws.Endpoint, error) {
				if storageSpec.S3.Region != "" {
					region = storageSpec.S3.Region
				}
				return aws.Endpoint{
					PartitionID:       "aws",
					URL:               storageSpec.S3.Endpoint,
					SigningRegion:     region,
					HostnameImmutable: true,
				}, nil
			})
	}
	s3Client := s3.NewFromConfig(s3Config)

	client = &s3ClientWrapper{
		s3:         s3Client,
		downloader: manager.NewDownloader(s3Client),
		endpoint:   storageSpec.S3.Endpoint,
		region:     storageSpec.S3.Region,
	}
	s.clients[clientIdentifier] = client
	return client
}

func (s *StorageProvider) explodeKey(ctx context.Context, client *s3ClientWrapper, bucket, key string) ([]s3ObjectSummary, error) {
	if !strings.HasSuffix(key, "*") {
		headResp, err := client.s3.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return nil, err
		}

		if headResp.ContentType != nil && !strings.HasPrefix(*headResp.ContentType, "application/x-directory") {
			return []s3ObjectSummary{{
				key:  aws.String(key),
				size: headResp.ContentLength,
			}}, nil
		}
	}

	// if the key is a directory, or ends with a wildcard, we need to list the objects starting with the key
	sanitizedKey := s.sanitizeKey(key)
	res := make([]s3ObjectSummary, 0)
	var continuationToken *string
	for {
		resp, err := client.s3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(bucket),
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
