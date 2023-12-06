package repo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// GitServer sets up a local git repo and then uses git daemon to
// allow tests to run against a local repository. This removes the
// need for tests to pull from somewhere like github.
//
// To use the git server you should create a temporary folder,
// and then another folder within that temporary folder. Like so..
//
// testFolder, _ := os.MkDirTemp("", "")
// projectFolder, _ := filepath.Join(testFolder, "fakeorg/project.git")
//
// If you add a file called helloworld.txt to project folder, you can
// use the server as:
//
// gs, _ := NewGitServer(testFolder, projectFolder)
// gs.Init("helloworld.txt")
// gs.Serve()
//
// You can check out helloworld.txt with
//
//	git clone git://127.0.0.1:9418/fakeorg/project.git
type GitServer struct {
	rootFolder    string
	projectFolder string
}

func NewGitServer(rootFolder string, projectFolder string) (*GitServer, error) {
	path, err := filepath.Abs(rootFolder)
	if err != nil {
		return nil, err
	}

	return &GitServer{
		rootFolder:    path,
		projectFolder: filepath.Join(rootFolder, projectFolder),
	}, nil
}

func (g *GitServer) runCommandInFolder(fldr string, cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Dir = fldr

	output, err := c.Output()
	fmt.Println(string(output))
	if err != nil {
		return err
	}

	return nil
}

func (g *GitServer) Init(withFiles ...string) error {
	if err := g.runCommandInFolder(g.projectFolder, "git", "init", "."); err != nil {
		return err
	}

	// Write magic file 'git-daemon-export-ok' to the newly created .git
	// folder
	target := filepath.Join(g.projectFolder, ".git", "git-daemon-export-ok")
	if f, err := os.OpenFile(target, os.O_RDONLY|os.O_CREATE, 0666); err != nil {
		return err
	} else {
		f.Close()
	}

	for _, f := range withFiles {
		if err := g.AddFile(f); err != nil {
			return err
		}
	}

	return nil
}

// AddFiles adds a file (that already exists in g.folder) to the git
// repository
func (g *GitServer) AddFile(file string) error {
	if err := g.runCommandInFolder(g.projectFolder, "git", "add", file); err != nil {
		return err
	}
	return g.runCommandInFolder(g.projectFolder, "git", "commit", file, "-m", "'added'")
}

func (g *GitServer) Serve(cancellableCtx context.Context) (*exec.Cmd, error) {
	c := exec.CommandContext(cancellableCtx, "git", "daemon", "--reuseaddr", fmt.Sprintf("--base-path=%s", g.rootFolder), g.rootFolder)
	c.Dir = g.rootFolder

	if err := c.Start(); err != nil {
		return nil, err
	}

	return c, nil
}
