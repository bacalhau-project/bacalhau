package iroh

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	iroh "github.com/n0-computer/iroh-ffi/iroh"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type Client struct {
	node   *iroh.IrohNode
	author *iroh.AuthorId
}

func New(path string) (*Client, error) {
	node, err := iroh.NewIrohNode(path)
	if err != nil {
		return nil, fmt.Errorf("creating iroh node at path %s: %w", path, err)
	}
	author, err := node.AuthorNew()
	if err != nil {
		return nil, fmt.Errorf("creating iroh node author: %w", err)
	}
	return &Client{
		node:   node,
		author: author,
	}, nil
}

func (c *Client) IsInstalled(ctx context.Context) (bool, error) {
	return c.node != nil, nil
}

func (c *Client) ValidateJob(ctx context.Context, j models.Job) error {
	if j.Task().Publisher.Type == "iroh" {
		return nil
	}
	return fmt.Errorf("invalid publisher type: %s expected iroh", j.Task().Publisher.Type)
}

/*

This is all from the number 0 discord

Todo this correctly:
 how to add data from a dir https://discord.com/channels/949724860232392765/1171477759416082544/1171574380770377800
 how to download the data to a dir: https://discord.com/channels/949724860232392765/1171477759416082544/1171584421556650135
for now will just tar it.

my process for doing locally:
$ cd iroh-ffi
$ git checkout b5/dall-e-example-fixes
$ ./make_go.sh
$ cd ../iroh-examples/dall_e_worker
$ export LD_LIBRARY_PATH="${LD_LIBRARY_PATH:-}:/path/to/iroh-ffi/target/debug"
$ export CGO_LDFLAGS="-liroh -L /path/to/iroh-ffi/target/debug"
$ export OPENAI_API_KEY="your_secret_api_key"
$ go run main.go $IROH_TICKET

you can get an iroh ticket either from iroh.network (the "invite" button in the console), or by running iroh start locally, then in another terminal
$ iroh console
> doc create --switch
> doc share write
# ticket will output here

forrest — Today at 8:43 AM
what does the replace directive look like in you main.go? something like replace github.com/n0-computer/iroh-ffi => ../../n0-computer/iroh-ffi/go?
b5 — Today at 8:58 AM
yep exactly
mine:
replace github.com/n0-computer/iroh-ffi => ../iroh-ffi/go
I have iroh-ffi and iroh-examples as sibling directories

*/

func (c *Client) PublishResult(
	ctx context.Context,
	execution *models.Execution,
	resultPath string,
) (models.SpecConfig, error) {
	// adding a single file to Iroh is significantly easier. Tar it.
	var buf bytes.Buffer
	if err := Tar(resultPath, &buf); err != nil {
		return models.SpecConfig{}, fmt.Errorf("tarring result path: %w", err)
	}

	doc, err := c.node.DocNew()
	if err != nil {
		return models.SpecConfig{}, fmt.Errorf("creating iroh document: %w", err)
	}
	hash, err := doc.SetBytes(c.author, []byte("results"), buf.Bytes())
	if err != nil {
		return models.SpecConfig{}, fmt.Errorf("writing results at %s to doc %s: %w", resultPath, doc.Id().ToString(), err)
	}

	readTicket, err := doc.Share(iroh.ShareModeRead)
	if err != nil {
		log.Error().Err(err).Msg("sharing doc failed")
		//return models.SpecConfig{}, fmt.Errorf("sharing doc %s: %w", doc.Id().ToString(), err)
	}

	var ticket string
	if readTicket != nil {
		ticket = readTicket.ToString()
	}
	return models.SpecConfig{
		Type: models.StorageSourceIroh,
		Params: map[string]interface{}{
			"doc":    doc.Id().ToString(),
			"ticket": ticket,
			"hash":   hash.ToString(),
		},
	}, nil
}

func (c *Client) DescribeResult(ctx context.Context, result model.PublishedResult) (map[string]string, error) {
	return nil, fmt.Errorf("not supported")
}

func (c *Client) FetchResult(ctx context.Context, item model.DownloadItem) error {
	hash, err := iroh.HashFromString(item.Metadata["hash"])
	if err != nil {
		return fmt.Errorf("decoding hash from string: %w", err)
	}

	ticket, err := iroh.DocTicketFromString(item.Metadata["ticket"])
	if err != nil {
		return fmt.Errorf("parsing doc ticket %s: %w", item.Metadata["ticket"], err)
	}

	doc, err := c.node.DocJoin(ticket)
	if err != nil {
		return fmt.Errorf("joinning doc: %w", err)
	}
	defer doc.Leave()

	// TODO I don't think I need to call doc.Sync here...I hope
	//doc.StartSync()

	// TODO this is all happening in memory :'(
	data, err := c.node.BlobsReadToBytes(hash)
	if err != nil {
		return fmt.Errorf("reading blob from hash %s: %w", hash.ToString(), err)
	}

	if err := Untar(item.Target, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("untarring data: %w", err)
	}

	return err
}

// Tar takes a source and variable writers and walks 'source' writing each file
// found to the tar writer; the purpose for accepting multiple writers is to allow
// for multiple outputs (for example a file, or md5 hash)
func Tar(src string, writers ...io.Writer) error {

	// ensure the src actually exists before trying to tar it
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("unable to tar files - %w", err)
	}

	mw := io.MultiWriter(writers...)

	gzw := gzip.NewWriter(mw)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	// walk path
	return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {

		// return on any error
		if err != nil {
			return err
		}

		// return on non-regular files (thanks to [kumo](https://medium.com/@komuw/just-like-you-did-fbdd7df829d3) for this suggested update)
		if !fi.Mode().IsRegular() {
			return nil
		}

		// create a new dir/file header
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		// update the name to correctly reflect the desired destination when untaring
		header.Name = strings.TrimPrefix(strings.Replace(file, src, "", -1), string(filepath.Separator))

		// write the header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// open files for taring
		f, err := os.Open(file)
		if err != nil {
			return err
		}

		// copy file data into tar writer
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		// manually close here after each file operation; defering would cause each file close
		// to wait until all operations have completed.
		f.Close()

		return nil
	})
}

// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func Untar(dst string, r io.Reader) error {

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}
}
