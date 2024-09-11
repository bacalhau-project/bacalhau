package webui

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

//go:embed build/**
var buildFiles embed.FS

func ListenAndServe(ctx context.Context, host, apiPort, apiPath string, listenPort int) error {
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", listenPort),
		Handler:           http.HandlerFunc(serveFiles),
		ReadTimeout:       time.Minute,
		WriteTimeout:      time.Minute,
		ReadHeaderTimeout: time.Minute,
		IdleTimeout:       time.Minute,
		BaseContext:       func(l net.Listener) context.Context { return ctx },
	}

	log.Printf("Starting server on port %d", listenPort)
	return server.ListenAndServe()
}

func serveFiles(w http.ResponseWriter, r *http.Request) {
	// Adjust the requested path to look inside the 'build' directory
	// This assumes all our static files are in a 'build' subdirectory
	fsPath := path.Join("build", strings.TrimPrefix(r.URL.Path, "/"))

	// Attempt to open the file at the computed path
	file, err := buildFiles.Open(fsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// If the file doesn't exist, we serve our custom 404 page
			serve404(w, r)
			return
		}
		// For any other kind of error, log it and return a 500 error
		log.Printf("Error opening file %s: %v", fsPath, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Get the file info to check if it's a directory and for modification time
	stat, err := file.Stat()
	if err != nil {
		log.Printf("Error getting file info for %s: %v", fsPath, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if stat.IsDir() {
		// If the path is a directory, we try to serve an index.html file from that directory
		indexPath := path.Join(fsPath, "index.html")
		indexFile, err := buildFiles.Open(indexPath)
		if err == nil {
			// If we found an index.html in this directory, serve it
			defer indexFile.Close()
			serveFileContent(w, r, indexFile, "index.html")
			return
		}
		// If there's no index.html in this specific directory,
		// or if we encountered any errors, serve our custom 404 page
		serve404(w, r)
		return
	}

	// If we've reached here, we're dealing with a normal file (not a directory)
	// Serve the file with its correct name
	serveFileContent(w, r, file, stat.Name())
}

// serveFileContent reads the entire file into memory and serves it
// This approach is used because fs.File doesn't guarantee implementation of io.ReadSeeker
func serveFileContent(w http.ResponseWriter, r *http.Request, file fs.File, name string) {
	// Read the entire file content
	content, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Error reading file %s: %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Create a ReadSeeker from the content and serve it
	// We use time.Time{} as the modtime, which will make the file always downloadable
	// You might want to get the actual modtime if caching is important
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(content))
}

// serve404 serves the custom 404.html file
// This is used as a fallback for non-existent paths or directories without an index.html
func serve404(w http.ResponseWriter, r *http.Request) {
	notFoundPath := path.Join("build", "404.html")
	notFoundFile, err := buildFiles.Open(notFoundPath)
	if err != nil {
		// If we can't find the 404.html, fall back to standard NotFound response
		log.Printf("404 page not found: %s", notFoundPath)
		http.NotFound(w, r)
		return
	}
	defer notFoundFile.Close()

	// Read the content of the 404 page
	content, err := io.ReadAll(notFoundFile)
	if err != nil {
		log.Printf("Error reading 404 page: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set the status code and content type
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Write the content
	_, err = w.Write(content)
	if err != nil {
		log.Printf("Error writing 404 page: %v", err)
	}
}
