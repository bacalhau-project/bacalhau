package ipfs_http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/filecoin-project/bacalhau/pkg/system"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
	archiver "github.com/mholt/archiver/v4"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/rs/zerolog/log"
)

type IPFSHttpClient struct {
	ctx     context.Context
	Address string
	Api     *httpapi.HttpApi
}

func NewIPFSHttpClient(
	ctx context.Context,
	address string,
) (*IPFSHttpClient, error) {
	addr, err := ma.NewMultiaddr(address)
	if err != nil {
		return nil, err
	}
	api, err := httpapi.NewApi(addr)
	if err != nil {
		return nil, err
	}
	return &IPFSHttpClient{
		ctx:     ctx,
		Address: address,
		Api:     api,
	}, nil
}

func (ipfsHttp *IPFSHttpClient) GetLocalAddrs() ([]ma.Multiaddr, error) {
	return ipfsHttp.Api.Swarm().LocalAddrs(ipfsHttp.ctx)
}

func (ipfsHttp *IPFSHttpClient) GetPeers() ([]iface.ConnectionInfo, error) {
	return ipfsHttp.Api.Swarm().Peers(ipfsHttp.ctx)
}

func (ipfsHttp *IPFSHttpClient) GetLocalAddrStrings() ([]string, error) {
	addressStrings := []string{}
	addrs, err := ipfsHttp.GetLocalAddrs()
	if err != nil {
		return addressStrings, nil
	}
	for _, addr := range addrs {
		addressStrings = append(addressStrings, addr.String())
	}
	return addressStrings, nil
}

// the libp2p addresses we should connect to
func (ipfsHttp *IPFSHttpClient) GetSwarmAddresses() ([]string, error) {
	addressStrings := []string{}
	addresses, err := ipfsHttp.GetLocalAddrStrings()
	if err != nil {
		return addressStrings, nil
	}
	peerId, err := ipfsHttp.GetPeerId()
	if err != nil {
		return addressStrings, nil
	}
	for _, address := range addresses {
		addressStrings = append(addressStrings, fmt.Sprintf("%s/p2p/%s", address, peerId))
	}
	return addressStrings, nil
}

func (ipfsHttp *IPFSHttpClient) GetPeerId() (string, error) {
	key, err := ipfsHttp.Api.Key().Self(ipfsHttp.ctx)
	if err != nil {
		return "", err
	}
	return key.ID().String(), nil
}

// return the peer ids of peers that provide the given cid
func (ipfsHttp *IPFSHttpClient) GetCidProviders(cid string) ([]string, error) {
	peerChan, err := ipfsHttp.Api.Dht().FindProviders(ipfsHttp.ctx, path.New(cid))
	if err != nil {
		return []string{}, err
	}
	providers := []string{}
	for addressInfo := range peerChan {
		providers = append(providers, addressInfo.ID.String())
	}
	return providers, nil
}

func (ipfsHttp *IPFSHttpClient) HasCidLocally(cid string) (bool, error) {
	peerId, err := ipfsHttp.GetPeerId()
	if err != nil {
		return false, err
	}
	providers, err := ipfsHttp.GetCidProviders(cid)
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

func (ipfsHttp *IPFSHttpClient) DownloadTar(cid, targetDir string) error {
	res, err := ipfsHttp.Api.
		Request("get", cid).
		Send(ipfsHttp.ctx)
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
	err = system.RunCommand("tar", []string{
		"-vxf", tarfilePath, "-C", targetDir,
	})
	if err != nil {
		return err
	}
	log.Debug().Msgf("Extracted tar file: %s", tarfilePath)
	os.Remove(tarfilePath)
	if err != nil {
		return err
	}
	return nil
}

func (ipfsHttp *IPFSHttpClient) UploadTar(sourceDir string) (string, error) {
	contentType, body, err := createUploadForm(sourceDir)
	if err != nil {
		return "", err
	}
	url, err := ipfsHttp.GetUrl()
	if err != nil {
		return "", err
	}
	resp, err := http.Post(
		fmt.Sprintf("%s/api/v0/add?pin=true&recursive=true&wrap-with-directory=true", url),
		contentType,
		body,
	)
	if err != nil {
		return "", err
	}
	respAsBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	fmt.Printf("HAVE RESULT FROM TAR UPLOAD:\n%s\n", string(respAsBytes))
	return string(respAsBytes), nil
}

func createUploadForm(folderPath string) (string, io.Reader, error) {
	files, err := archiver.FilesFromDisk(nil, map[string]string{
		folderPath: "",
	})
	if err != nil {
		return "", nil, err
	}
	format := archiver.CompressedArchive{
		Compression: archiver.Gz{},
		Archival:    archiver.Tar{},
	}
	body := new(bytes.Buffer)
	mp := multipart.NewWriter(body)
	formFieldWriter, err := mp.CreateFormFile("file", "results.tar")
	if err != nil {
		return "", nil, err
	}
	err = format.Archive(context.Background(), formFieldWriter, files)
	if err != nil {
		return "", nil, err
	}
	return mp.FormDataContentType(), body, nil
}
