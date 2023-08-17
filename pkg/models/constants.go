package models

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

const (
	StorageSourceNoop           = "noop"
	StorageSourceIPFS           = "ipfs"
	StorageSourceRepoClone      = "repoClone"
	StorageSourceRepoCloneLFS   = "repoCloneLFS"
	StorageSourceEstuary        = "estuary"
	StorageSourceURL            = "urlDownload"
	StorageSourceS3             = "s3"
	StorageSourceInline         = "inline"
	StorageSourceLocalDirectory = "localDirectory"
)

const (
	PublisherNoop    = "noop"
	PublisherIPFS    = "ipfs"
	PublisherEstuary = "estuary"
	PublisherS3      = "s3"
)

const (
	DownloadFilenameStdout   = "stdout"
	DownloadFilenameStderr   = "stderr"
	DownloadFilenameExitCode = "exitCode"
	DownloadCIDsFolderName   = "raw"
	DownloadFolderPerm       = 0755
	DownloadFilePerm         = 0644
)

const (
	MetaRequesterID        = "bacalhau.org/requester.id"
	MetaRequesterPublicKey = "bacalhau.org/requester.publicKey"
	MetaClientID           = "bacalhau.org/client.id"
)

const (
	ShortIDLength = 8
)
