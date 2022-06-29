package ipfshttp

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/system"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

type IPFSHTTPClient struct {
	Address string
	API     *httpapi.HttpApi
}

func NewIPFSHTTPClient(address string) (*IPFSHTTPClient, error) {
	addr, err := ma.NewMultiaddr(address)
	if err != nil {
		return nil, err
	}
	api, err := httpapi.NewApi(addr)
	if err != nil {
		return nil, err
	}
	return &IPFSHTTPClient{
		Address: address,
		API:     api,
	}, nil
}

func (ipfsHTTP *IPFSHTTPClient) GetLocalAddrs(ctx context.Context) ([]ma.Multiaddr, error) {
	ctx, span := newSpan(ctx, "GetLocalAddrs")
	defer span.End()

	return ipfsHTTP.API.Swarm().LocalAddrs(ctx)
}

func (ipfsHTTP *IPFSHTTPClient) GetPeers(ctx context.Context) ([]iface.ConnectionInfo, error) {
	ctx, span := newSpan(ctx, "GetPeers")
	defer span.End()

	return ipfsHTTP.API.Swarm().Peers(ctx)
}

func (ipfsHTTP *IPFSHTTPClient) GetLocalAddrStrings(ctx context.Context) ([]string, error) {
	ctx, span := newSpan(ctx, "GetLocalAddrStrings")
	defer span.End()

	addressStrings := []string{}
	addrs, err := ipfsHTTP.GetLocalAddrs(ctx)
	if err != nil {
		return addressStrings, nil
	}

	for _, addr := range addrs {
		addressStrings = append(addressStrings, addr.String())
	}

	return addressStrings, nil
}

// the libp2p addresses we should connect to
func (ipfsHTTP *IPFSHTTPClient) GetSwarmAddresses(ctx context.Context) ([]string, error) {
	ctx, span := newSpan(ctx, "GetSwarmAddresses")
	defer span.End()

	addressStrings := []string{}
	addresses, err := ipfsHTTP.GetLocalAddrStrings(ctx)
	if err != nil {
		return nil, err
	}

	peerID, err := ipfsHTTP.GetPeerID(ctx)
	if err != nil {
		return nil, err
	}

	for _, address := range addresses {
		addressStrings = append(addressStrings, fmt.Sprintf("%s/p2p/%s", address, peerID))
	}

	return addressStrings, nil
}

func (ipfsHTTP *IPFSHTTPClient) GetPeerID(ctx context.Context) (string, error) {
	ctx, span := newSpan(ctx, "GetPeerId")
	defer span.End()

	key, err := ipfsHTTP.API.Key().Self(ctx)
	if err != nil {
		return "", err
	}

	return key.ID().String(), nil
}

// return the peer ids of peers that provide the given cid
func (ipfsHTTP *IPFSHTTPClient) GetCidProviders(ctx context.Context, cid string) ([]string, error) {
	ctx, span := newSpan(ctx, "GetCidProviders")
	defer span.End()

	peerChan, err := ipfsHTTP.API.Dht().FindProviders(ctx, path.New(cid))
	if err != nil {
		return []string{}, err
	}

	providers := []string{}
	for addressInfo := range peerChan {
		providers = append(providers, addressInfo.ID.String())
	}

	return providers, nil
}

func (ipfsHTTP *IPFSHTTPClient) HasCidLocally(ctx context.Context, cid string) (bool, error) {
	ctx, span := newSpan(ctx, "HasCidLocally")
	defer span.End()

	peerID, err := ipfsHTTP.GetPeerID(ctx)
	if err != nil {
		return false, err
	}

	providers, err := ipfsHTTP.GetCidProviders(ctx, cid)
	if err != nil {
		return false, err
	}

	return system.StringArrayContains(providers, peerID), nil
}

func (ipfsHTTP *IPFSHTTPClient) GetURL() (string, error) {
	addr, err := ma.NewMultiaddr(ipfsHTTP.Address)
	if err != nil {
		return "", err
	}

	_, url, err := manet.DialArgs(addr)
	if err != nil {
		return "", err
	}

	if a, err := ma.NewMultiaddr(url); err == nil {
		_, host, err := manet.DialArgs(a)
		if err == nil {
			url = host
		}
	}

	return url, nil
}

func (ipfsHTTP *IPFSHTTPClient) DownloadTar(ctx context.Context, targetDir, cid string) error {
	ctx, span := newSpan(ctx, "DownloadTar")
	defer span.End()

	res, err := ipfsHTTP.API.Request("get", cid).Send(ctx)
	if err != nil {
		return err
	}
	defer res.Close()

	tarfilePath := fmt.Sprintf("%s/%s.tar", targetDir, cid)
	log.Debug().Msgf("Writing cid: %s tar file to %s", cid, tarfilePath)

	outFile, err := os.Create(tarfilePath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, res.Output)
	if err != nil {
		return err
	}

	_, err = system.RunCommandGetResults("tar", []string{
		"-vxf", tarfilePath, "-C", targetDir,
	})
	if err != nil {
		return err
	}

	log.Debug().Msgf("Extracted tar file: %s", tarfilePath)
	os.Remove(tarfilePath)

	return nil
}

// TODO: #291 we need to work out how to upload a tar file
// using just the HTTP api and not needing to shell out
func (ipfsHTTP *IPFSHTTPClient) UploadTar(ctx context.Context, sourceDir string) (string, error) {
	_, span := newSpan(ctx, "UploadTar")
	defer span.End()

	result, err := system.RunCommandGetResults("ipfs", []string{
		"--api", ipfsHTTP.Address,
		"add", "-rq", sourceDir,
	})
	if err != nil {
		return "", err
	}

	parts := strings.Split(result, "\n")
	if len(parts) <= 1 {
		return "", fmt.Errorf("no parts returned from ipfs add")
	}

	return parts[len(parts)-2], nil
}

func newSpan(ctx context.Context, api string) (context.Context, trace.Span) {
	return system.Span(ctx, "ipfs/http", api)
}
