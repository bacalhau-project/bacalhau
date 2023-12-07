//go:build unit || !integration

package urldownload

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/configenv"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type StorageSuite struct {
	suite.Suite
	RootCmd *cobra.Command
}

func TestStorageSuite(t *testing.T) {
	err := config.Set(configenv.Local)
	require.NoError(t, err)
	suite.Run(t, new(StorageSuite))
}

// Before each test
func (s *StorageSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
}

func (s *StorageSuite) TestHasStorageLocally() {
	sp := NewStorage()

	spec := models.InputSource{
		Source: &models.SpecConfig{
			Type: models.StorageSourceURL,
			Params: Source{
				URL: "foo",
			}.ToMap(),
		},
		Target: "bar",
	}
	// files are not cached thus shall never return true
	locally, err := sp.HasStorageLocally(context.Background(), spec)
	s.Require().NoError(err, "failed to check if storage is locally available")

	s.False(locally, "storage should not be locally available")
}

func (s *StorageSuite) TestPrepareStorageURL() {
	type dummyRequest struct {
		path    string
		code    int
		content string
		headers *map[string]string
	}
	tests := []struct {
		name             string
		requests         []dummyRequest
		expectedContent  string
		expectedFilename string
	}{
		{
			name: "follows-redirect",
			requests: []dummyRequest{
				{
					path:    "/initial",
					code:    302,
					content: "/second.png",
				},
				{
					path:    "/second.png",
					code:    302,
					content: "/third.txt",
				},
				{
					path:    "/third.txt",
					code:    200,
					content: "this is from the final redirect",
				},
			},
			expectedContent:  "this is from the final redirect",
			expectedFilename: "third.txt",
		},
		{
			name: "retries",
			requests: []dummyRequest{
				{
					path:    "/initial",
					code:    500,
					content: "",
				},
				{
					path:    "/initial",
					code:    500,
					content: "",
				},
				{
					path:    "/initial",
					code:    200,
					content: "got there eventually",
				},
			},
			expectedContent:  "got there eventually",
			expectedFilename: "initial",
		},
		{
			name: "retry-anything",
			requests: []dummyRequest{
				{
					path:    "/initial",
					code:    401,
					content: "not allowed",
				},
				{
					path:    "/initial",
					code:    401,
					content: "not allowed",
				},
				{
					path:    "/initial",
					code:    200,
					content: "changed my mind",
				},
			},
			expectedContent:  "changed my mind",
			expectedFilename: "initial",
		},
		{
			name: "generates-name",
			requests: []dummyRequest{
				{
					path:    "/",
					code:    200,
					content: "name should be a UUID",
				},
			},
			expectedContent:  "name should be a UUID",
			expectedFilename: "",
		},
		{
			name: "no-content",
			requests: []dummyRequest{
				{
					path:    "/nothing.txt",
					code:    204,
					content: "",
				},
			},
			expectedContent:  "",
			expectedFilename: "nothing.txt",
		},
		{
			name: "picsum.photos",
			requests: []dummyRequest{
				{
					path:    "/200/300",
					code:    302,
					content: "/id/568/200/300.jpg",
				},
				{
					path:    "/id/568/200/300.jpg",
					code:    200,
					content: "i'm not putting an image here",
				},
			},
			expectedContent:  "i'm not putting an image here",
			expectedFilename: "300.jpg",
		},
		{
			name: "redirects.r.us - without content disposition",
			requests: []dummyRequest{
				{
					path:    "/img/300.jpg",
					code:    302,
					content: "/cdn/300/",
				},
				{
					path:    "/cdn/300/",
					code:    200,
					content: "i'm not putting an image here",
				},
			},
			expectedContent:  "i'm not putting an image here",
			expectedFilename: "300",
		},
		{
			name: "redirects.r.us - with content disposition",
			requests: []dummyRequest{
				{
					path:    "/img/300.jpg",
					code:    302,
					content: "/cdn/300/",
				},
				{
					path:    "/cdn/300/",
					code:    200,
					content: "i'm not putting an image here",
					headers: &map[string]string{
						"content-disposition": "attachment; filename*=UTF-8''300.jpg; filename=\"300.jpg\";",
					},
				},
			},
			expectedContent:  "i'm not putting an image here",
			expectedFilename: "300.jpg",
		},
		{
			name: "redirects.r.us - malicious",
			requests: []dummyRequest{
				{
					path:    "/img/300.jpg",
					code:    302,
					content: "/cdn/300/",
				},
				{
					path:    "/cdn/300/",
					code:    200,
					content: "i'm not putting an image here",
					headers: &map[string]string{
						"content-disposition": "attachment; filename*=UTF-8''300.jpg; filename=\"../../300.jpg\";",
					},
				},
			},
			expectedContent:  "i'm not putting an image here",
			expectedFilename: "300.jpg",
		},
		{
			name: "redirects.r.us - malicious part II",
			requests: []dummyRequest{
				{
					path:    "/img/300.jpg",
					code:    302,
					content: "/cdn/300/",
				},
				{
					path:    "/cdn/300/",
					code:    200,
					content: "i'm not putting an image here",
					headers: &map[string]string{
						"content-disposition": "attachment; filename*=UTF-8''300.jpg; filename=\"/etc/300.jpg\";",
					},
				},
			},
			expectedContent:  "i'm not putting an image here",
			expectedFilename: "300.jpg",
		},
		{
			name: "redirects.r.us - without redirect",
			requests: []dummyRequest{
				{
					path:    "/img/300.jpg",
					code:    200,
					content: "i'm not putting an image here",
					headers: &map[string]string{
						"content-disposition": "attachment; filename*=UTF-8''300.jpg; filename=\"dodgy.bin\";",
					},
				},
			},
			expectedContent:  "i'm not putting an image here",
			expectedFilename: "300.jpg",
		},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			responseCount := 0
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := test.requests[responseCount]
				responseCount++

				if r.URL.Path != response.path {
					http.Error(w, fmt.Sprintf("invalid path: %s should be %s", r.URL.Path, response.path), 999)
					return
				}

				if response.code == http.StatusFound {
					http.Redirect(w, r, response.content, http.StatusFound)
					return
				}

				if response.headers != nil {
					// Set the headers, if any, before WriteHeader is called
					for k, v := range *response.headers {
						w.Header().Add(k, v)
					}
				}

				w.WriteHeader(response.code)

				_, err := w.Write([]byte(response.content))
				s.NoError(err)
			}))
			s.T().Cleanup(ts.Close)

			subject := NewStorage()

			url := fmt.Sprintf("%s%s", ts.URL, test.requests[0].path)
			spec := models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceURL,
					Params: Source{
						URL: url,
					}.ToMap(),
				},
				Target: "/inputs",
			}

			vol, err := subject.PrepareStorage(context.Background(), s.T().TempDir(), spec)
			s.Require().NoError(err)

			actualFilename := filepath.Base(vol.Source)
			if test.expectedFilename == "" {
				s.Regexp(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`, actualFilename,
					"Filename should be a UUID if it can't come from the HTTP response ")
				s.Regexp(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`, filepath.Base(vol.Target))
				s.Equal(fmt.Sprintf("%s%s", string(os.PathSeparator), "inputs"), filepath.Dir(vol.Target))
			} else {
				s.Equal(test.expectedFilename, actualFilename, fmt.Sprintf("filename doesn't match calling %s", url))
				s.Equal(filepath.Join("/inputs", test.expectedFilename), vol.Target)
			}

			s.FileExists(vol.Source)
			actualContent, err := os.ReadFile(vol.Source)
			s.Require().NoError(err)

			s.Equal(test.expectedContent, string(actualContent), "content does not match")
		})
	}
}
