package s3

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	sync "github.com/bacalhau-project/golang-mutex-tracer"
)

type ClientWrapper struct {
	S3         *s3.Client
	Downloader *manager.Downloader
	Uploader   *manager.Uploader
	Endpoint   string
	Region     string
}

type ClientProviderParams struct {
	AWSConfig aws.Config
}

type ClientProvider struct {
	awsConfig aws.Config
	clients   map[string]*ClientWrapper
	mu        sync.RWMutex
}

func NewClientProvider(params ClientProviderParams) *ClientProvider {
	c := &ClientProvider{
		awsConfig: params.AWSConfig,
		clients:   make(map[string]*ClientWrapper),
	}

	c.mu.EnableTracerWithOpts(sync.Opts{
		Threshold: 50 * time.Millisecond,
		Id:        "S3SClientProvider.mu",
	})
	return c
}

// IsInstalled returns true if the S3 client is installed.
func (s *ClientProvider) IsInstalled() bool {
	return HasValidCredentials(s.awsConfig)
}

// GetConfig returns the AWS config used by the client provider.
func (s *ClientProvider) GetConfig() aws.Config {
	return s.awsConfig
}

func (s *ClientProvider) GetClient(endpoint, region string) *ClientWrapper {
	clientIdentifier := fmt.Sprintf("%s-%s", endpoint, region)
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
	if region != "" {
		s3Config.Region = region
	}
	if endpoint != "" {
		s3Config.EndpointResolverWithOptions =
			aws.EndpointResolverWithOptionsFunc(func(service, region2 string, options ...any) (aws.Endpoint, error) {
				if region2 != "" {
					region2 = region
				}
				return aws.Endpoint{
					PartitionID:       "aws",
					URL:               endpoint,
					SigningRegion:     region2,
					HostnameImmutable: true,
				}, nil
			})
	}
	s3Client := s3.NewFromConfig(s3Config)

	client = &ClientWrapper{
		S3:         s3Client,
		Downloader: manager.NewDownloader(s3Client),
		Uploader:   manager.NewUploader(s3Client),
		Endpoint:   endpoint,
		Region:     region,
	}
	s.clients[clientIdentifier] = client
	return client
}
