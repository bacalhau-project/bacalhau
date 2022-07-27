package urldownload

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"net/http"
	"net/http/httptest"
	"io/ioutil"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/storage"
)

func TestNewStorageProvider(t *testing.T) {
	cm := system.NewCleanupManager()
	sp, err := NewStorageProvider(cm)
	if err != nil {
		t.Fatal(err)
	}
	// is dir writable?
	fmt.Println(sp.LocalDir)
	f, err := os.Create(filepath.Join(sp.LocalDir, "data.txt"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("test\n")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if sp.HTTPClient == nil {
		t.Error("HTTP client in StorageProvider is nil")
	}
}


func TestHasStorageLocally(t *testing.T) {
	cm := system.NewCleanupManager()
	sp, err := NewStorageProvider(cm)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	spec := storage.StorageSpec{
		Engine: storage.StorageSourceURLDownload,
		URL:    "foo",
		Path:   "foo",
	}
	// files are not cached thus shall never return true
	locally, err := sp.HasStorageLocally(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	if locally != false {
		t.Error("StorageProvider should not have any files stored locally")
	}
}

func TestPrepareStorage(t *testing.T) {
	testString := "Here's your data"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == "/testfile" {
			w.Write([]byte(testString))
		}
	}))
	defer ts.Close()

	cm := system.NewCleanupManager()
	sp, err := NewStorageProvider(cm)
	if err != nil {
		t.Fatal(err)
	}

	serverURL := ts.URL
	spec := storage.StorageSpec{
		Engine: storage.StorageSourceURLDownload,
		URL:    serverURL + "/testfile",
		Path:   "/foo",
	}

	ctx := context.Background()
	volume, err := sp.PrepareStorage(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	
	file, err := os.Open(volume.Source)
	if err != nil {
        t.Fatal(err)
    }
	defer func() {
        if err = file.Close(); err != nil {
            t.Fatal(err)
        }
    }()

	content, err := ioutil.ReadAll(file)
	if err != nil {
        t.Fatal(err)
    }
	text := string(content)
	if text != testString {
		t.Errorf("Should be \"%s\", but is \"%s\"", testString, text)
	}
}
