package webui

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

//go:embed build/**
var buildFiles embed.FS

type Config struct {
	APIEndpoint string `json:"APIEndpoint"`
	Listen      string `json:"Listen"`
}

type Server struct {
	apiURL        *url.URL
	listenAddress string
	mux           *http.ServeMux
	apiProxy      *httputil.ReverseProxy
}

func NewServer(cfg Config) (*Server, error) {
	if cfg.Listen == "" {
		return nil, fmt.Errorf("listen address cannot be empty")
	}
	// Parse and validate API endpoint
	apiURL, err := normalizeAPIEndpoint(cfg.APIEndpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid API endpoint: %w", err)
	}

	s := &Server{
		apiURL:        apiURL,
		listenAddress: cfg.Listen,
		mux:           http.NewServeMux(),
		apiProxy:      newReverseProxy(apiURL),
	}

	s.mux.HandleFunc("/api/", s.handleAPIProxy)
	s.mux.HandleFunc("/", s.handleFiles)

	return s, nil
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	server := &http.Server{
		Addr:              s.listenAddress,
		Handler:           s.mux,
		ReadTimeout:       time.Minute,
		WriteTimeout:      time.Minute,
		ReadHeaderTimeout: time.Minute,
		IdleTimeout:       time.Minute,
		BaseContext:       func(l net.Listener) context.Context { return ctx },
	}

	// Graceful shutdown
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("Server shutdown error")
		}
	}()

	log.Info().Str("listen", server.Addr).Str("backend", sanitizeURL(s.apiURL)).Msg("Starting server")
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func (s *Server) handleAPIProxy(w http.ResponseWriter, r *http.Request) {
	log.Trace().
		Str("path", r.URL.Path).
		Str("method", r.Method).
		Bool("is_websocket", r.Header.Get("Upgrade") == "websocket").
		Msg("Proxying request")
	s.apiProxy.ServeHTTP(w, r)
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	statusCode := http.StatusOK
	var message string
	var attemptedPaths []string

	defer func() {
		duration := time.Since(startTime)
		logEvent := log.With().
			Str("path", r.URL.Path).
			Int("status", statusCode).
			Dur("duration", duration)

		if len(attemptedPaths) > 0 {
			logEvent = logEvent.Strs("attempted_paths", attemptedPaths)
		}

		logger := logEvent.Logger()

		switch {
		case statusCode >= 500:
			logger.Error().Msg(message)
		case statusCode == http.StatusNotFound:
			logger.Warn().Msg(message)
		default:
			logger.Trace().Msg(message)
		}
	}()

	// Adjust the requested path to look inside the 'build' directory
	fsPath := path.Join("build", strings.TrimPrefix(r.URL.Path, "/"))
	attemptedPaths = append(attemptedPaths, fsPath)

	// Attempt to open the file at the computed path
	file, err := buildFiles.Open(fsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// If the file doesn't exist, try with .html extension
			htmlPath := fsPath + ".html"
			attemptedPaths = append(attemptedPaths, htmlPath)
			file, err = buildFiles.Open(htmlPath)
			if err != nil {
				// If still not found, serve our custom 404 page
				statusCode = http.StatusNotFound
				message = "File not found"
				s.serve404(w, r)
				return
			}
			fsPath = htmlPath
		} else {
			// For any other kind of error, log it and return a 500 error
			statusCode = http.StatusInternalServerError
			message = fmt.Sprintf("Failed to open file: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
	_ = defer file.Close()

	// Get the file info to check if it's a directory and for modification time
	stat, err := file.Stat()
	if err != nil {
		statusCode = http.StatusInternalServerError
		message = fmt.Sprintf("Failed to get file info: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if stat.IsDir() {
		// If the path is a directory, we try to serve an index.html file from that directory
		indexPath := path.Join(fsPath, "index.html")
		attemptedPaths = append(attemptedPaths, indexPath)
		indexFile, err := buildFiles.Open(indexPath)
		if err == nil {
			// If we found an index.html in this directory, serve it
			_ = defer indexFile.Close()
			message = "Served index.html from directory"
			s.serveFileContent(w, r, indexFile, "index.html")
			return
		}
		// If there's no index.html in this specific directory,
		// or if we encountered any errors, serve our custom 404 page
		statusCode = http.StatusNotFound
		message = "Directory without index.html"
		s.serve404(w, r)
		return
	}

	// If we've reached here, we're dealing with a normal file (not a directory)
	// Serve the file with its correct name
	message = fmt.Sprintf("Served file: %s", stat.Name())
	s.serveFileContent(w, r, file, stat.Name())
}

// serveFileContent reads the entire file into memory and serves it
// This approach is used because fs.File doesn't guarantee implementation of io.ReadSeeker
func (s *Server) serveFileContent(w http.ResponseWriter, r *http.Request, file fs.File, name string) {
	// Read the entire file content
	content, err := io.ReadAll(file)
	if err != nil {
		log.Error().Err(err).Str("filename", name).Msg("Failed to read file content")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Create a ReadSeeker from the content and serve it
	// We use time.Time{} as the modtime, which will make the file always downloadable
	// You might want to get the actual modtime if caching is important
	http.ServeContent(w, r, name, time.Time{}, strings.NewReader(string(content)))
}

// serve404 serves the custom 404.html file
// This is used as a fallback for non-existent paths or directories without an index.html
func (s *Server) serve404(w http.ResponseWriter, r *http.Request) {
	notFoundPath := path.Join("build", "404.html")
	notFoundFile, err := buildFiles.Open(notFoundPath)
	if err != nil {
		// If we can't find the 404.html, fall back to standard NotFound response
		log.Warn().Str("path", notFoundPath).Msg("Custom 404 page not found")
		http.NotFound(w, r)
		return
	}
	_ = defer notFoundFile.Close()

	// Read the content of the 404 page
	content, err := io.ReadAll(notFoundFile)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read 404 page content")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set the status code and content type
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Write the content
	if _, err := w.Write(content); err != nil {
		log.Error().Err(err).Msg("Failed to write 404 page content")
	}
}

// normalizeAPIEndpoint validates and normalizes the API endpoint URL
func normalizeAPIEndpoint(endpoint string) (*url.URL, error) {
	if !strings.HasPrefix(strings.ToLower(endpoint), "http") {
		endpoint = "http://" + endpoint
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme %q", u.Scheme)
	}

	if u.Host == "" {
		return nil, fmt.Errorf("missing host")
	}

	u.Path = strings.TrimSuffix(u.Path, "/")
	return u, nil
}

func newReverseProxy(target *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		response := map[string]string{
			"error":    "Backend service unavailable",
			"endpoint": sanitizeURL(target),
			"method":   r.Method,
			"path":     r.URL.Path,
		}

		log.Error().Err(err).Any("response", response).Msg("Proxy error")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(response)
	}
	return proxy
}

// sanitizeURL removes sensitive information from URL for error messages
func sanitizeURL(u *url.URL) string {
	sanitized := *u
	sanitized.User = nil
	sanitized.RawQuery = ""
	sanitized.Fragment = ""
	return sanitized.String()
}
