package models

import (
	"math"
	"time"
)

var NoTimeout = time.Duration(math.MaxInt64).Truncate(time.Second)

const (
	// DefaultNamespace is the default namespace.
	DefaultNamespace = "default"
)

const (
	// JobTypeService represents a long-running job that runs on a desired number of nodes
	// matching the specified constraints.
	JobTypeService = "service"

	// JobTypeDaemon represents a long-running job that runs on all nodes matching the
	// specified constraints.
	JobTypeDaemon = "daemon"

	// JobTypeBatch represents a batch job that runs to completion on the desired number
	// of nodes matching the specified constraints.
	JobTypeBatch = "batch"

	// JobTypeOps represents a batch job that runs to completion on all nodes matching
	// the specified constraints.
	JobTypeOps = "ops"
)

const (
	EngineNoop   = "noop"
	EngineDocker = "docker"
	EngineWasm   = "wasm"
)

var EngineNames = []string{
	EngineDocker,
	EngineWasm,
}

const (
	StorageSourceNoop           = "noop"
	StorageSourceIPFS           = "ipfs"
	StorageSourceURL            = "urlDownload"
	StorageSourceS3             = "s3"
	StorageSourceS3PreSigned    = "s3PreSigned"
	StorageSourceInline         = "inline"
	StorageSourceLocalDirectory = "localDirectory"
)

var StoragesNames = []string{
	StorageSourceIPFS,
	StorageSourceInline,
	StorageSourceLocalDirectory,
	StorageSourceS3,
	StorageSourceS3PreSigned,
	StorageSourceURL,
}

const (
	PublisherNoop  = "noop"
	PublisherIPFS  = "ipfs"
	PublisherS3    = "s3"
	PublisherLocal = "local"
)

var PublisherNames = []string{
	PublisherNoop,
	PublisherIPFS,
	PublisherS3,
	PublisherLocal,
}

const (
	DownloadFilenameStdout   = "stdout"
	DownloadFilenameStderr   = "stderr"
	DownloadFilenameExitCode = "exitCode"
	DownloadCIDsFolderName   = "raw"
	DownloadFolderPerm       = 0755
	DownloadFilePerm         = 0644
)

const (
	MetaReservedPrefix       = "bacalhau.org/"
	MetaOrchestratorIDLegacy = "bacalhau.org/requester.id"
	MetaOrchestratorID       = "bacalhau.org/orchestrator.id"
	// MetaOrchestratorProtocol indicates which orchestrator-compute protocol is used
	MetaOrchestratorProtocol = "bacalhau.org/orchestrator.protocol"

	MetaServerInstallationID = "bacalhau.org/server.installation.id"
	MetaServerInstanceID     = "bacalhau.org/server.instance.id"
	MetaClientInstallationID = "bacalhau.org/client.installation.id"
	MetaClientInstanceID     = "bacalhau.org/client.instance.id"
)
