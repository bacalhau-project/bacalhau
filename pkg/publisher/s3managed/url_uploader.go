package s3managed

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

// Upload uploads a file to the specified URL
type URLUploader interface {
	Upload(ctx context.Context, url string, filePath string) error
}

const (
	PreSignedURLUploadRetryCount = 5
)

type S3PreSignedURLUploader struct {
	httpClient *http.Client
	retryCount int
}

// NewS3PreSignedURLUploader creates a new uploader that uses pre-signed URLs to upload files to S3.
func NewS3PreSignedURLUploader(httpClient *http.Client) URLUploader {
	return &S3PreSignedURLUploader{
		httpClient: httpClient,
		retryCount: PreSignedURLUploadRetryCount,
	}
}

func (u *S3PreSignedURLUploader) Upload(ctx context.Context, url string, filePath string) error {
	return u.uploadWithRetry(ctx, url, filePath, u.retryCount)
}

func (u *S3PreSignedURLUploader) uploadWithPreSignedURL(
	ctx context.Context,
	filePath string,
	presignedURL string,
) error {
	file, err := os.Open(filePath) //nolint:gosec // G304: filePath from uploader config, application controlled
	if err != nil {
		return fmt.Errorf("failed to open file for upload: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Get file info for logging
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file stats: %w", err)
	}

	// Log the upload attempt
	log.Ctx(ctx).Debug().
		Str("file", filePath).
		Int64("size", fileInfo.Size()).
		Msg("Uploading file to S3 using pre-signed URL")

	// Create PUT request using the pre-signed URL
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, presignedURL, file)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	// Set content type and length headers for the upload
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = fileInfo.Size()

	// Perform the upload
	resp, err := u.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("upload failed with status code: %d", resp.StatusCode)
	}

	return nil
}

func (u *S3PreSignedURLUploader) uploadWithRetry(
	ctx context.Context,
	url string,
	filePath string,
	retryCount int,
) error {
	var lastErr error
	for attempt := 0; attempt <= retryCount; attempt++ {
		if attempt > 0 {
			// Wait before retrying, with exponential backoff
			backoff := attempt - 1
			if backoff < 0 {
				backoff = 0
			}
			backoffTime := time.Duration(1<<backoff) * time.Second
			if backoffTime > 30*time.Second {
				backoffTime = 30 * time.Second
			}

			log.Ctx(ctx).Debug().
				Str("file", filePath).
				Int("attempt", attempt).
				Int("maxRetries", retryCount).
				Dur("backoffTime", backoffTime).
				Err(lastErr).
				Msg("Retrying upload after failure")

			select {
			case <-time.After(backoffTime):
				// Continue after backoff
			case <-ctx.Done():
				// Context cancelled, abort retries
				return ctx.Err()
			}
		}

		// Attempt upload
		err := u.uploadWithPreSignedURL(ctx, filePath, url)
		if err == nil {
			// Success
			if attempt > 0 {
				log.Ctx(ctx).Debug().
					Str("file", filePath).
					Int("attempt", attempt+1).
					Msg("Upload succeeded after retries")
			}
			return nil
		}

		// Save error for potential retry
		lastErr = err
	}

	return fmt.Errorf("failed to upload after %d attempts: %w", retryCount+1, lastErr)
}
