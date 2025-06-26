package s3managed_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/s3managed"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockURLUploader mocks the URLUploader interface
type MockURLUploader struct {
	ShouldFail bool
	UploadURL  string
	FilePath   string
}

func (m *MockURLUploader) Upload(ctx context.Context, url string, filePath string) error {
	m.UploadURL = url
	m.FilePath = filePath

	if m.ShouldFail {
		return fmt.Errorf("upload failed")
	}
	return nil
}

// MockPublisher implements the ncl.Publisher interface
type MockPublisher struct {
	ShouldFail      bool
	PreSignedURL    string
	ReceivedRequest ncl.PublishRequest
}

func (m *MockPublisher) Request(ctx context.Context, req ncl.PublishRequest) (*envelope.Message, error) {
	m.ReceivedRequest = req

	if m.ShouldFail {
		return nil, fmt.Errorf("failed to get presigned URL")
	}

	// Extract job ID and execution ID from request
	msgPayload, ok := req.Message.Payload.(messages.ManagedPublisherPreSignURLRequest)
	if !ok {
		return nil, fmt.Errorf("unexpected payload type")
	}

	// Create response with pre-signed URL
	response := &messages.ManagedPublisherPreSignURLResponse{
		JobID:        msgPayload.JobID,
		ExecutionID:  msgPayload.ExecutionID,
		PreSignedURL: m.PreSignedURL,
	}

	// envelope.NewMessage already returns a pointer to a message
	return envelope.NewMessage(response), nil
}

func (m *MockPublisher) Publish(ctx context.Context, req ncl.PublishRequest) error {
	return nil
}

// MockPublisherProvider implements the ncl.PublisherProvider interface
type MockPublisherProvider struct {
	Publisher  ncl.Publisher
	ShouldFail bool
}

func (m *MockPublisherProvider) GetPublisher() (ncl.Publisher, error) {
	if m.ShouldFail {
		return nil, fmt.Errorf("failed to get publisher")
	}
	return m.Publisher, nil
}

// Tests

func TestIsInstalled(t *testing.T) {
	// Managed S3 publisher is always considered installed
	publisher := s3managed.NewPublisher(s3managed.PublisherParams{})
	installed, err := publisher.IsInstalled(context.Background())

	assert.NoError(t, err)
	assert.True(t, installed)
}

func TestValidateJob(t *testing.T) {
	publisher := s3managed.NewPublisher(s3managed.PublisherParams{})

	t.Run("success", func(t *testing.T) {
		// Create a job with S3 Managed publisher type
		job := mock.Job()
		job.Tasks[0].Publisher.Type = models.PublisherS3Managed

		err := publisher.ValidateJob(context.Background(), *job)
		assert.NoError(t, err)
	})

	t.Run("failure", func(t *testing.T) {
		// Create a job with invalid publisher type
		job := mock.Job()
		job.Tasks[0].Publisher.Type = "invalid-publisher"

		err := publisher.ValidateJob(context.Background(), *job)
		assert.Error(t, err)
	})
}

func TestPublishResult_Success(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	resultPath := filepath.Join(tmpDir, "results")
	require.NoError(t, os.MkdirAll(resultPath, 0755))

	// Create a test file to simulate execution results
	testFile := filepath.Join(resultPath, "output.txt")
	testContent := []byte("Test execution result")
	require.NoError(t, os.WriteFile(testFile, testContent, 0644))

	// Create mocks
	mockUploader := &MockURLUploader{}
	mockPreSignedURL := "https://test-bucket.s3.amazonaws.com/job123/exec456?X-Amz-Signature=abcdef"
	mockNCLPublisher := &MockPublisher{
		PreSignedURL: mockPreSignedURL,
	}
	mockProvider := &MockPublisherProvider{
		Publisher: mockNCLPublisher,
	}

	// Create publisher
	publisher := s3managed.NewPublisher(s3managed.PublisherParams{
		LocalDir:             tmpDir,
		URLUploader:          mockUploader,
		NCLPublisherProvider: mockProvider,
	})

	// Create job and execution
	job := mock.Job()
	job.ID = "job123"
	execution := &models.Execution{
		ID:  "exec456",
		Job: job,
	}

	// Test
	result, err := publisher.PublishResult(context.Background(), execution, resultPath)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, models.StorageSourceS3Managed, result.Type)
	assert.Equal(t, "job123", result.Params["JobID"])
	assert.Equal(t, "exec456", result.Params["ExecutionID"])

	// Verify the upload was called with the correct URL
	assert.Equal(t, mockPreSignedURL, mockUploader.UploadURL)
	assert.NotEmpty(t, mockUploader.FilePath)
}

func TestPublishResult_NclPublisherError(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	resultPath := filepath.Join(tmpDir, "results")
	require.NoError(t, os.MkdirAll(resultPath, 0755))

	// Create mocks with error condition
	mockUploader := &MockURLUploader{}
	mockProvider := &MockPublisherProvider{
		ShouldFail: true,
	}

	// Create publisher
	publisher := s3managed.NewPublisher(s3managed.PublisherParams{
		LocalDir:             tmpDir,
		URLUploader:          mockUploader,
		NCLPublisherProvider: mockProvider,
	})

	// Create job and execution
	job := mock.Job()
	job.ID = "job123"
	execution := &models.Execution{
		ID:  "exec456",
		Job: job,
	}

	// Test
	_, err := publisher.PublishResult(context.Background(), execution, resultPath)

	// Verify
	require.Error(t, err)
}

func TestPublishResult_UploadError(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	resultPath := filepath.Join(tmpDir, "results")
	require.NoError(t, os.MkdirAll(resultPath, 0755))

	// Create a test file
	testFile := filepath.Join(resultPath, "output.txt")
	testContent := []byte("Test execution result")
	require.NoError(t, os.WriteFile(testFile, testContent, 0644))

	// Create mocks with upload error
	mockUploader := &MockURLUploader{
		ShouldFail: true,
	}
	mockPreSignedURL := "https://test-bucket.s3.amazonaws.com/job123/exec456?X-Amz-Signature=abcdef"
	mockNCLPublisher := &MockPublisher{
		PreSignedURL: mockPreSignedURL,
	}
	mockProvider := &MockPublisherProvider{
		Publisher: mockNCLPublisher,
	}

	// Create publisher
	publisher := s3managed.NewPublisher(s3managed.PublisherParams{
		LocalDir:             tmpDir,
		URLUploader:          mockUploader,
		NCLPublisherProvider: mockProvider,
	})

	// Create job and execution
	job := mock.Job()
	job.ID = "job123"
	execution := &models.Execution{
		ID:  "exec456",
		Job: job,
	}

	// Test
	_, err := publisher.PublishResult(context.Background(), execution, resultPath)

	// Verify
	require.Error(t, err)
}
