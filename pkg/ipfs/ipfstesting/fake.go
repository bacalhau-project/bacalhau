package ipfstesting

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ipfs/boxo/files"
	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	mh "github.com/multiformats/go-multihash"

	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

var (
	// HashFunction is the default hash function for computing CIDs.
	//
	// This is currently Blake2b-256.
	HashFunction = uint64(mh.BLAKE2B_MIN + 31)

	// When producing a CID for an IPLD block less than or equal to CIDInlineLimit
	// bytes in length, the identity hash function will be used instead of
	// HashFunction. This will effectively "inline" the block into the CID, allowing
	// it to be extracted directly from the CID with no disk/network operations.
	//
	// This is currently -1 for "disabled".
	//
	// This is exposed for testing. Do not modify unless you know what you're doing.
	CIDInlineLimit = -1
)

type cidBuilder struct {
	codec uint64
}

func (cidBuilder) WithCodec(c uint64) cid.Builder {
	return cidBuilder{codec: c}
}

func (b cidBuilder) GetCodec() uint64 {
	return b.codec
}

func (b cidBuilder) Sum(data []byte) (cid.Cid, error) {
	hf := HashFunction
	if len(data) <= CIDInlineLimit {
		hf = mh.IDENTITY
	}
	return cid.V1Builder{Codec: b.codec, MhType: hf}.Sum(data)
}

// CidBuilder is the default CID builder for Filecoin.
//
// - The default codec is CBOR. This can be changed with CidBuilder.WithCodec.
// - The default hash function is 256bit blake2b.
var CidBuilder cid.Builder = cidBuilder{codec: cid.DagCBOR}

func makeCID(input string, prefix *cid.Prefix) cid.Cid {
	data := []byte(input)
	if prefix == nil {
		c, err := CidBuilder.Sum(data)
		if err != nil {
			panic(err)
		}
		return c
	}
	c, err := prefix.Sum(data)
	switch {
	case errors.Is(err, mh.ErrSumNotSupported):
		// multihash library doesn't support this hash function.
		// just fake it.
	case err == nil:
		return c
	default:
		panic(err)
	}

	return c
}

func NewFakeIPFSNode() ipfs.Node {
	return &fakeIPFSNode{data: make(map[cid.Cid]string)}
}

type fakeIPFSNode struct {
	data map[cid.Cid]string
}

func (f *fakeIPFSNode) ID(ctx context.Context) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (f *fakeIPFSNode) APIAddress() string {
	return "fakeIPFSNode"
}

type fakeFileNode struct {
	size int64
}

func (f *fakeFileNode) Close() error {
	return nil
}

func (f *fakeFileNode) Size() (int64, error) {
	return f.size, nil
}

func (f *fakeIPFSNode) Get(ctx context.Context, c cid.Cid) (files.Node, error) {
	data, ok := f.data[c]
	if !ok {
		return nil, fmt.Errorf("content %s not found", c)
	}
	return &fakeFileNode{size: int64(len(data))}, nil
}

func (f *fakeIPFSNode) Size(ctx context.Context, c cid.Cid) (uint64, error) {
	data, ok := f.data[c]
	if !ok {
		return 0, fmt.Errorf("content %s not found", c)
	}
	return uint64(len(data)), nil
}

func (f *fakeIPFSNode) Has(ctx context.Context, c cid.Cid) (bool, error) {
	_, ok := f.data[c]
	return ok, nil
}

func (f *fakeIPFSNode) Put(ctx context.Context, path string) (cid.Cid, error) {
	c, err := CidBuilder.Sum([]byte(path))
	if err != nil {
		return cid.Undef, err
	}
	f.data[c] = path
	return c, nil
}

func (f *fakeIPFSNode) GetTreeNode(ctx context.Context, cid cid.Cid) (ipfs.IPLDTreeNode, error) {
	//TODO implement me
	panic("implement me")
}

func AddFileToNodes(ctx context.Context, filePath string, client ipfs.Node) (string, error) {
	var res string
	cid, err := client.Put(ctx, filePath)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("error adding %q to node ", filePath))
	}

	log.Ctx(ctx).Debug().Msgf("Added CID %q to IPFS node %q", cid, client.APIAddress())
	res = strings.TrimSpace(cid.String())

	return res, nil
}

func AddTextToNodes(ctx context.Context, fileContent []byte, client ipfs.Node) (string, error) {
	tempDir, err := os.MkdirTemp("", "bacalhau-test")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempDir)

	testFilePath := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFilePath, fileContent, util.OS_USER_RW|util.OS_ALL_R)
	if err != nil {
		return "", err
	}

	return AddFileToNodes(ctx, testFilePath, client)
}
