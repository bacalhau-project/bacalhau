package s3

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type ClientWrapper struct {
	S3            *s3.Client
	presignClient *s3.PresignClient
	Downloader    *manager.Downloader
	Uploader      *manager.Uploader
	Endpoint      string
	Region        string
	mu            sync.RWMutex
}

// RecalculateV4Signature struct and its methods
type RecalculateV4Signature struct {
	next   http.RoundTripper
	signer *v4.Signer
	cfg    aws.Config
}

func (lt *RecalculateV4Signature) RoundTrip(req *http.Request) (*http.Response, error) {
	// Store for later use
	val := req.Header.Get("Accept-Encoding")

	// Delete the header so it doesn't account for in the signature
	req.Header.Del("Accept-Encoding")

	// Sign with the same date
	timeString := req.Header.Get("X-Amz-Date")
	timeDate, _ := time.Parse("20060102T150405Z", timeString)

	creds, _ := lt.cfg.Credentials.Retrieve(req.Context())
	err := lt.signer.SignHTTP(req.Context(), creds, req, v4.GetPayloadHash(req.Context()), "s3", lt.cfg.Region, timeDate)
	if err != nil {
		return nil, err
	}

	// Reset Accept-Encoding if needed
	req.Header.Set("Accept-Encoding", val)

	return lt.next.RoundTrip(req)
}

func (c *ClientWrapper) PresignClient() *s3.PresignClient {
	c.mu.RLock()
	if c.presignClient != nil {
		defer c.mu.RUnlock()
		return c.presignClient
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.presignClient != nil {
		return c.presignClient
	}
	c.presignClient = s3.NewPresignClient(c.S3)
	return c.presignClient
}

type ClientProviderParams struct {
	AWSConfig aws.Config
}

type ClientProvider struct {
	awsConfig aws.Config
	clients   map[string]*ClientWrapper
	clientsMu sync.RWMutex
}

func NewClientProvider(params ClientProviderParams) *ClientProvider {
	return &ClientProvider{
		awsConfig: params.AWSConfig,
		clients:   make(map[string]*ClientWrapper),
	}
}

func (s *ClientProvider) IsInstalled() bool {
	return HasValidCredentials(s.awsConfig)
}

func (s *ClientProvider) GetConfig() aws.Config {
	return s.awsConfig
}

func (s *ClientProvider) GetClient(endpoint, region string) *ClientWrapper {
	clientIdentifier := fmt.Sprintf("%s-%s", endpoint, region)
	s.clientsMu.RLock()
	client, ok := s.clients[clientIdentifier]
	s.clientsMu.RUnlock()
	if ok {
		return client
	}

	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

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
			aws.EndpointResolverWithOptionsFunc(func(service, resolvedRegion string, options ...any) (aws.Endpoint, error) {
				if region != "" {
					resolvedRegion = region
				}
				return aws.Endpoint{
					PartitionID:       "aws",
					URL:               endpoint,
					SigningRegion:     resolvedRegion,
					HostnameImmutable: true,
				}, nil
			})
	}
	// Set the custom HTTP client with signature recalculating logic
	if strings.Contains(endpoint, "https://storage.googleapis.com") {
		s3Config.EndpointResolverWithOptions = aws.EndpointResolverWithOptionsFunc(func(service, resolvedRegion string, options ...any) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               endpoint,
				SigningRegion:     "auto",
				Source:            aws.EndpointSourceCustom,
				HostnameImmutable: true,
			}, nil
		})
		s3Config.Region = "auto"
		s3Config.Credentials = credentials.NewStaticCredentialsProvider(os.Getenv("GCP_ACCESS_KEY_ID"), os.Getenv("GCP_SECRET_ACCESS_KEY"), "session")
		s3Config.HTTPClient = &http.Client{
			Transport: &RecalculateV4Signature{
				next:   http.DefaultTransport,
				signer: v4.NewSigner(),
				cfg:    s3Config,
			}}
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
