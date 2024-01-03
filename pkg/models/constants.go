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
	StorageSourceRepoClone      = "repoclone"
	StorageSourceRepoCloneLFS   = "repoCloneLFS"
	StorageSourceURL            = "urlDownload"
	StorageSourceS3             = "s3"
	StorageSourceS3PreSigned    = "s3PreSigned"
	StorageSourceInline         = "inline"
	StorageSourceLocalDirectory = "localDirectory"
)

const (
	PublisherNoop = "noop"
	PublisherIPFS = "ipfs"
	PublisherS3   = "s3"
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
	MetaReservedPrefix = "bacalhau.org/"
	MetaRequesterID    = "bacalhau.org/requester.id"
	MetaClientID       = "bacalhau.org/client.id"

	// Job provenance metadata used to track the origin of a job where
	// it may have been translated from another job.
	MetaDerivedFrom  = "bacalhau.org/derivedFrom"
	MetaTranslatedBy = "bacalhau.org/translatedBy"
)
