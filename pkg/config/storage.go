package config

import (
	"os"
	"path"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/viper"
)

const DirectoryPerms = 0755

const (
	executionsPath = "executor_storages"
	resultsPath    = "compute_store"
	pluginsPath    = "plugins"
	natsPath       = "nats"
)

type computeNodeStorage struct{}
type requesterNodeStorage struct {
	nats string
}

type nodeStorage interface {
	computeNodeStorage | requesterNodeStorage
}

// NodeStorage is a struct that holds the root path for the storage of a node
// and the inner storage type which differentiates from compute and requester nodes.
// This is unfortunately only for documentation purposes as it isn't possible to
// constrain functions to the inner type. This means that if something is
//
//	`(n *NodeStorage[requesterNodeStorage])`
//
// it can still be called by NodeStorage[computeNodeStorage] as they are all
// actually just NodeStorage[nodeStorage].
type NodeStorage[T nodeStorage] struct {
	rootPath string
	inner    T
}

func GetComputeNodeStorage(optionalRoot ...string) (*NodeStorage[computeNodeStorage], error) {
	var path string

	if len(optionalRoot) == 1 {
		path = optionalRoot[0]
	} else {
		path = viper.GetString(types.NodeStoragePath)
		if path == "" {
			path = os.TempDir()
		}
	}

	n := &NodeStorage[computeNodeStorage]{
		rootPath: path,
		inner:    computeNodeStorage{},
	}

	err := n.init()
	return n, err
}

func GetRequesterNodeStorage(optionalRoot ...string) (*NodeStorage[requesterNodeStorage], error) {
	var path string

	if len(optionalRoot) == 1 {
		path = optionalRoot[0]
	} else {
		path = viper.GetString(types.NodeStoragePath)
		if path == "" {
			path = os.TempDir()
		}
	}

	n := &NodeStorage[requesterNodeStorage]{
		rootPath: path,
		inner:    requesterNodeStorage{},
	}

	err := n.init()
	return n, err
}

// initialises the storage paths to make sure the roots we need for various
// components are present (or created) ahead of time.
func (n *NodeStorage[T]) init() error {
	switch any(n.inner).(type) {
	case computeNodeStorage:
		return n.ensureComputeStorage()
	case requesterNodeStorage:
		return n.ensureRequesterStorage()
	}
	return nil // should never happen
}

// Ensures the paths needed for compute node are present
func (n *NodeStorage[computeNodeStorage]) ensureComputeStorage() error {
	var errs *multierror.Error

	for _, pth := range []string{executorsPath, resultsPath, pluginsPath} {
		p := filepath.Join(n.rootPath, pth)
		errs = multierror.Append(errs, n.ensureDirExists(p))
	}

	return errs.ErrorOrNil()
}

func (n *NodeStorage[requesterNodeStorage]) ensureRequesterStorage() error {
	errs := new(multierror.Error)

	for _, pth := range []string{natsPath} {
		p := filepath.Join(n.rootPath, pth)
		errs = multierror.Append(errs, n.ensureDirExists(p))
	}

	return errs.ErrorOrNil()
}

func (n *NodeStorage[nodeStorage]) ensureDirExists(path string) error {
	return os.MkdirAll(path, DirectoryPerms)
}

func (n *NodeStorage[nodeStorage]) GetRoot() string {
	return n.rootPath
}

func (n *NodeStorage[computeNodeStorage]) GetExecutionStoragePath() string {
	return path.Join(n.rootPath, executorsPath)
}

func (n *NodeStorage[computeNodeStorage]) GetResultsStoragePath() string {
	return path.Join(n.rootPath, resultsPath)
}

func (n *NodeStorage[computeNodeStorage]) GetPluginStoragePath() string {
	return path.Join(n.rootPath, pluginsPath)
}

func (n *NodeStorage[requesterNodeStorage]) GetNATSStoragePath() string {
	return path.Join(n.rootPath, natsPath)
}
