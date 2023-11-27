package webui

import (
	"bytes"
	"context"
	"embed"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"time"
)

// This file exists to expose the Web UI components to the Bacalhau app.

//go:embed build/**
var buildFiles embed.FS

// Return the index page for the web app with the passed API URL used. The app
// running on each client will attempt to connect to the API URL to retrieve job
// and node information.
func IndexPage(host, port, base string) ([]byte, error) {
	rawIndex, err := buildFiles.ReadFile("build/index.html")
	if err != nil {
		return nil, err
	}

	indexTemplate, err := template.New("script").Parse(string(rawIndex))
	if err != nil {
		return nil, err
	}

	var output bytes.Buffer
	err = indexTemplate.Execute(&output, struct{ Host, Port, Base string }{host, port, base})
	return output.Bytes(), err
}

func ListenAndServe(ctx context.Context, host, port, base string) error {
	indexPage, err := IndexPage(host, port, base)
	if err != nil {
		return err
	}

	files, err := fs.Sub(buildFiles, "build")
	if err != nil {
		return err
	}

	fileHandler := http.FileServer(http.FS(files))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		match, err := fs.Glob(files, strings.TrimLeft(r.URL.Path, "/"))
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Write(indexPage)
		} else if len(match) > 0 && err == nil {
			// Serve a static file that matches the request path, if it exists
			// We need to check this manually to enable the below behavior
			fileHandler.ServeHTTP(w, r)
		} else {
			// If nothing matches, just serve the index page. This allows the
			// app to modify the browser URL to refer to client side "pages"
			// (e.g. `/jobs`) which the user can then successfully refresh,
			// share or bookmark. If this wasn't here, the user would have to
			// enter the app manually at `/` every time.
			w.Write(indexPage)
		}
	})

	server := &http.Server{
		Handler:           &handler,
		ReadTimeout:       time.Minute,
		WriteTimeout:      time.Minute,
		ReadHeaderTimeout: time.Minute,
		IdleTimeout:       time.Minute,
		BaseContext:       func(l net.Listener) context.Context { return ctx },
	}

	return server.ListenAndServe()
}
