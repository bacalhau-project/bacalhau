package clone

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os/exec"
	"path"
	"path/filepath"
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

// func Clone(url string)Clone{
// 	hash:=URLtoLatestCommitHash(url)
// 	c := clone{
// 		SHA1HASH:hash,
// 		CID:""

// 	}

// }

func NewCloneClient() (*Clone, error) {

	return &Clone{
		URL: "",
	}, nil
}

func (cl *Clone) CreateSHA1CIDPair(key string, value string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// add form fields
	writer.WriteField("key", key)
	writer.WriteField("value", value)

	err := writer.Close()
	if err != nil {
		fmt.Println(err)
		return err
	}

	// create request
	req, err := http.NewRequest("POST", "http://kv.bacalhau.org/", body)
	if err != nil {
		fmt.Println(err)
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// send request
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR")

		fmt.Println(err)
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		log.Println("Request successful")
	} else {
		log.Println("Request failed")
	}
	// print response
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("ERROR")
		fmt.Println(err)
		return err
	}
	fmt.Println(string(data))
	return nil
}

func (cl *Clone) RunShellScript(scriptPath string, args []string) (string, error) {
	scriptArgs := []string{}
	scriptArgs = append(scriptArgs, scriptPath)
	scriptArgs = append(scriptArgs, args...)

	if len(args) > 0 {

		last := scriptArgs[len(scriptArgs)-1]
		path, err := ioutil.TempDir("", path.Base(last))
		scriptArgs[len(scriptArgs)-1] = path
		if err != nil {

			fmt.Println(err)

		}
		return path, nil
	}

	x := fmt.Sprintf("/bin/bash %v", scriptArgs)
	fmt.Print(x)
	cmd := exec.Command(x)
	err := cmd.Start()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("Waiting for %s script to finish...", scriptPath)
	err = cmd.Wait()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Script finished")
	// Run the command and get the output
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println()
		return "", fmt.Errorf("failed to execute script: %s", err)

	}
	fmt.Println()

	// Print the output
	fmt.Printf("%s\n", output)

	return "", nil
}

func (cl *Clone) IfNotInstalledInstallingGitlfs() {
	_, err := filepath.Abs("./install-lfs.sh")

	// Printing if there is no error
	if err != nil {
		fmt.Printf("can't find the script: %v", err)
	}
	args := ScriptStruct{
		path: "pkg/clone/install-lfs.sh", arguments: []string{}}

	if _, err := cl.RunShellScript(args.path, args.arguments); err != nil {
		fmt.Println(err)
	}

}

func (cl *Clone) CloneRepo(repoURL *url.URL, Path string) (string, error) {
	_, err := filepath.Abs("./clone.sh")

	// Printing if there is no error
	if err != nil {
		fmt.Printf("can't find the script: %v", err)
	}
	args := ScriptStruct{
		path:      "pkg/clone/clone.sh",
		arguments: []string{},
	}

	args.arguments = append(args.arguments, repoURL.String())
	args.arguments = append(args.arguments, Path)

	path, err := cl.RunShellScript(args.path, args.arguments)
	if err != nil {
		return "Error Cloning the repo", err
	} else {
		return path, nil
	}

}

func RepoExistsOnIPFSGivenUrl(urlStr string) (string, error) {
	cmd := exec.Command("git", "ls-remote", urlStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	x := "http://kv.bacalhau.org/" + fmt.Sprintf("%v", string(output)[:40])
	resp, _ := http.Get(x)

	if err != nil {
		return "", err

	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	// if err != nil {
	// 	return "", err

	// }
	// fmt.Println(string(body))

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err

	}
	return response.CID, nil

}

func (cl *Clone) UrltoLatestCommitHash(urlStr string) (string, error) {
	cmd := exec.Command("git", "ls-remote", urlStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	x := fmt.Sprintf("%v\n", string(output)[:40])
	return x, err
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
