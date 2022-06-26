package ipfs_cli

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type IPFSCli struct {
	Repo string
}

func NewIPFSCli(repo string) *IPFSCli {
	return &IPFSCli{
		Repo: repo,
	}
}

func (cli *IPFSCli) getBinaryFullPath() (string, error) {
	// TODO: We should have a struct that allows us to set the ipfs binary, rather than relying on system paths, etc
	ipfsBinary, err := exec.LookPath("ipfs")

	if err != nil {
		return "", fmt.Errorf("could not find 'ipfs' binary on your path")
	}

	ipfsBinaryFullPath, _ := filepath.Abs(ipfsBinary)

	if strings.Contains(ipfsBinaryFullPath, "/snap/") {
		return "", fmt.Errorf("you installed 'ipfs' using snap, which bacalhau is not compatible with. Please install from dist.ipfs.io or directly from your package provider") // nolint
	}

	return ipfsBinaryFullPath, nil
}

func (cli *IPFSCli) Run(args []string) (string, error) {

	ipfsBinaryFullPath, err := cli.getBinaryFullPath()

	if err != nil {
		return "", err
	}

	env := []string{}

	if cli.Repo != "" {
		env = append(env, "IPFS_PATH="+cli.Repo)
	}

	log.Trace().Msgf("ipfs command -->   IPFS_PATH=%s %s %s\n", cli.Repo, ipfsBinaryFullPath, strings.Join(args, " "))

	return system.RunCommandGetResultsEnv(ipfsBinaryFullPath, args, env)
}

func (cli *IPFSCli) HasCid(cid string) (bool, error) {
	allLocalRefString, err := cli.Run([]string{
		"refs",
		"local",
	})
	if err != nil {
		return false, err
	}
	log.Debug().Msgf("Comparing CID (%s) collecting to all refs in repo.", cid)
	allLocalRefsArray := strings.Split(allLocalRefString, "\n")
	log.Debug().Msgf("Total number of local refs: %d", len(allLocalRefsArray))
	if err != nil {
		return false, err
	}
	got := system.StringArrayContains(strings.Split(allLocalRefString, "\n"), cid)
	log.Debug().Msgf("CID (%s) in local refs: %t", cid, got)
	return got, nil
}

func (cli *IPFSCli) AddFolder(repoPath, folder string) (string, error) {
	allCidsString, err := cli.Run([]string{
		"add",
		"-rq",
		folder,
	})
	if err != nil {
		return "", err
	}
	allCids := strings.Split(allCidsString, "\n")
	// -2 is because it's the second last one before the newline
	return allCids[len(allCids)-2], nil
}
