package car

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-libipfs/blocks"
	"github.com/ipfs/go-unixfsnode/data/builder"
	"github.com/ipld/go-car/v2"
	"github.com/ipld/go-car/v2/blockstore"
	dagpb "github.com/ipld/go-codec-dagpb"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multicodec"
	"github.com/multiformats/go-multihash"
)

// copied from https://github.com/ipld/go-car/blob/master/cmd/car/create.go

func CreateCar(
	ctx context.Context,
	inputDirectory string,
	outputFile string,
	// this can be 1 or 2
	version int,
) (string, error) {
	// make a cid with the right length that we eventually will patch with the root.
	hasher, err := multihash.GetHasher(multihash.SHA2_256)
	if err != nil {
		return "", err
	}
	digest := hasher.Sum([]byte{})
	hash, err := multihash.Encode(digest, multihash.SHA2_256)
	if err != nil {
		return "", err
	}
	proxyRoot := cid.NewCidV1(uint64(multicodec.DagPb), hash)

	var options []car.Option

	switch version {
	case 1:
		options = []car.Option{blockstore.WriteAsCarV1(true)}
	case 2:
		// already the default
	default:
		return "", fmt.Errorf("invalid CAR version %d", version)
	}

	cdest, err := blockstore.OpenReadWrite(outputFile, []cid.Cid{proxyRoot}, options...)
	if err != nil {
		return "", err
	}

	// Write the unixfs blocks into the store.
	files, err := os.ReadDir(inputDirectory)
	if err != nil {
		return "", err
	}

	var carFilePaths []string
	for _, file := range files {
		carFilePaths = append(carFilePaths, fmt.Sprintf("%s/%s", inputDirectory, file.Name()))
	}

	root, err := writeFiles(ctx, cdest, carFilePaths...)
	if err != nil {
		return "", err
	}

	if err := cdest.Finalize(); err != nil {
		return "", err
	}
	// re-open/finalize with the final root.
	if err := car.ReplaceRootsInFile(outputFile, []cid.Cid{root}); err != nil {
		return "", err
	}
	return root.String(), nil
}

func writeFiles(ctx context.Context, bs *blockstore.ReadWrite, paths ...string) (cid.Cid, error) {
	ls := cidlink.DefaultLinkSystem()
	ls.TrustedStorage = true
	ls.StorageReadOpener = func(_ ipld.LinkContext, l ipld.Link) (io.Reader, error) {
		cl, ok := l.(cidlink.Link)
		if !ok {
			return nil, fmt.Errorf("not a cidlink")
		}
		blk, err := bs.Get(ctx, cl.Cid)
		if err != nil {
			return nil, err
		}
		return bytes.NewBuffer(blk.RawData()), nil
	}
	ls.StorageWriteOpener = func(_ ipld.LinkContext) (io.Writer, ipld.BlockWriteCommitter, error) {
		buf := bytes.NewBuffer(nil)
		return buf, func(l ipld.Link) error {
			cl, ok := l.(cidlink.Link)
			if !ok {
				return fmt.Errorf("not a cidlink")
			}
			blk, err := blocks.NewBlockWithCid(buf.Bytes(), cl.Cid)
			if err != nil {
				return err
			}
			bs.Put(ctx, blk) //nolint:errcheck
			return nil
		}, nil
	}

	topLevel := make([]dagpb.PBLink, 0, len(paths))
	for _, p := range paths {
		l, size, err := builder.BuildUnixFSRecursive(p, &ls)
		if err != nil {
			return cid.Undef, err
		}
		name := path.Base(p)
		entry, err := builder.BuildUnixFSDirectoryEntry(name, int64(size), l)
		if err != nil {
			return cid.Undef, err
		}
		topLevel = append(topLevel, entry)
	}

	// make a directory for the file(s).

	root, _, err := builder.BuildUnixFSDirectory(topLevel, &ls)
	if err != nil {
		return cid.Undef, nil
	}
	rcl, ok := root.(cidlink.Link)
	if !ok {
		return cid.Undef, fmt.Errorf("could not interpret %s", root)
	}

	return rcl.Cid, nil
}
