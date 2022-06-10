package ipfs_http

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
)

type IPFSHttpClient struct {
	Address string
	Api     *httpapi.HttpApi
}

func NewIPFSHttpClient(address string) (*IPFSHttpClient, error) {
	addr, err := ma.NewMultiaddr(address)
	if err != nil {
		return nil, err
	}
	api, err := httpapi.NewApi(addr)
	if err != nil {
		return nil, err
	}
	return &IPFSHttpClient{
		Address: address,
		Api:     api,
	}, nil
}

func (ipfsHttp *IPFSHttpClient) GetLocalAddrs(ctx context.Context) (
	[]ma.Multiaddr, error) {

	return ipfsHttp.Api.Swarm().LocalAddrs(ctx)
}

func (ipfsHttp *IPFSHttpClient) GetPeers(ctx context.Context) (
	[]iface.ConnectionInfo, error) {

	return ipfsHttp.Api.Swarm().Peers(ctx)
}

func (ipfsHttp *IPFSHttpClient) GetLocalAddrStrings(ctx context.Context) (
	[]string, error) {

	addressStrings := []string{}
	addrs, err := ipfsHttp.GetLocalAddrs(ctx)
	if err != nil {
		return addressStrings, nil
	}

	for _, addr := range addrs {
		addressStrings = append(addressStrings, addr.String())
	}

	return addressStrings, nil
}

// the libp2p addresses we should connect to
func (ipfsHttp *IPFSHttpClient) GetSwarmAddresses(ctx context.Context) (
	[]string, error) {

	addressStrings := []string{}
	addresses, err := ipfsHttp.GetLocalAddrStrings(ctx)
	if err != nil {
		return nil, err
	}

	peerId, err := ipfsHttp.GetPeerId(ctx)
	if err != nil {
		return nil, err
	}

	for _, address := range addresses {
		addressStrings = append(addressStrings, fmt.Sprintf("%s/p2p/%s", address, peerId))
	}

	return addressStrings, nil
}

func (ipfsHttp *IPFSHttpClient) GetPeerId(ctx context.Context) (string, error) {
	key, err := ipfsHttp.Api.Key().Self(ctx)
	if err != nil {
		return "", err
	}

	return key.ID().String(), nil
}

// return the peer ids of peers that provide the given cid
func (ipfsHttp *IPFSHttpClient) GetCidProviders(ctx context.Context,
	cid string) ([]string, error) {

	peerChan, err := ipfsHttp.Api.Dht().FindProviders(ctx, path.New(cid))
	if err != nil {
		return []string{}, err
	}

	providers := []string{}
	for addressInfo := range peerChan {
		providers = append(providers, addressInfo.ID.String())
	}

	return providers, nil
}

func (ipfsHttp *IPFSHttpClient) HasCidLocally(ctx context.Context,
	cid string) (bool, error) {

	peerId, err := ipfsHttp.GetPeerId(ctx)
	if err != nil {
		return false, err
	}

	providers, err := ipfsHttp.GetCidProviders(ctx, cid)
	if err != nil {
		return false, err
	}

	return system.StringArrayContains(providers, peerId), nil
}

func (ipfsHttp *IPFSHttpClient) GetUrl() (string, error) {
	addr, err := ma.NewMultiaddr(ipfsHttp.Address)
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

func (ipfsHttp *IPFSHttpClient) DownloadTar(ctx context.Context,
	targetDir, cid string) error {

	res, err := ipfsHttp.Api.Request("get", cid).Send(ctx)
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

// TODO: we need to work out how to upload a tar file
// using just the HTTP api and not needing to shell out
func (ipfsHttp *IPFSHttpClient) UploadTar(ctx context.Context,
	sourceDir string) (string, error) {

	result, err := system.RunCommandGetResults("ipfs", []string{
		"--api", ipfsHttp.Address,
		"add", "-rq", sourceDir,
	})
	if err != nil {
		return "", err
	}

	parts := strings.Split(result, "\n")
	if len(parts) <= 1 {
		return "", fmt.Errorf("No parts returned from ipfs add")
	}

	return parts[len(parts)-2], nil
}
