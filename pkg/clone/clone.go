package clone

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	//nolint:staticcheck
	"io/ioutil"
	"net/http"
	"net/url"
)

type ScriptStruct struct {
	path      string
	arguments []string
}

type Clone struct {
	URL string
}

type Response struct {
	CID string
}

func NewCloneClient() (*Clone, error) {
	return &Clone{
		URL: "",
	}, nil
}

func RepoExistsOnIPFSGivenURL(urlStr string) (string, error) {
	cmd := exec.Command("git", "ls-remote", urlStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	x := "http://kv.bacalhau.org/" + fmt.Sprintf("%v", string(output)[:40])
	//nolint:gosec,noctx
	resp, _ := http.Get(x)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}
	return response.CID, nil
}

func RemoveFromSlice(arr []string, item string) []string {
	newArr := []string{}
	for _, s := range arr {
		if s != item {
			newArr = append(newArr, s)
		}
	}
	return newArr
}

func IsValidGitRepoURL(urlStr string) (*url.URL, error) {
	// Check if the URL string is empty
	if urlStr == "" {
		return nil, fmt.Errorf("URL is empty")
	}
	// Parse the URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	// Check if the URL is a Git repository URL
	if u.Scheme != "https" && u.Scheme != "http" && u.Scheme != "ssh" {
		return nil, fmt.Errorf("URL must use HTTPS, HTTP, or SSH scheme")
	}
	if !strings.HasSuffix(u.Path, ".git") {
		return nil, fmt.Errorf("URL must use .git file extension")
	}
	return u, nil
}
