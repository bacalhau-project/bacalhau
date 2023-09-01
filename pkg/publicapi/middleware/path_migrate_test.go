//go:build unit || !integration

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestPathMigrate(t *testing.T) {
	// Define a map of old paths to new paths.
	paths := map[string]string{
		"/oldpath1": "/newpath1",
		"/oldpath2": "/newpath2",
	}

	// Setup a simple router with the PathMigrate middleware and a test route.
	r := chi.NewRouter()
	r.Use(PathMigrate(paths))
	r.Get("/newpath1", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("newpath1"))
	})
	r.Get("/newpath2", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("newpath2"))
	})

	tests := []struct {
		name           string
		inputPath      string
		expectedPath   string
		expectedBody   string
		expectedStatus int
	}{
		{
			name:           "Test old path 1 migration",
			inputPath:      "/oldpath1",
			expectedPath:   "/newpath1",
			expectedBody:   "newpath1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Test old path 2 migration",
			inputPath:      "/oldpath2",
			expectedPath:   "/newpath2",
			expectedBody:   "newpath2",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Test path not in map",
			inputPath:      "/pathNotInMap",
			expectedPath:   "/pathNotInMap",
			expectedBody:   "",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.inputPath, nil)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectedPath, req.URL.Path)
			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, tt.expectedBody, rr.Body.String())
			}
		})
	}
}
