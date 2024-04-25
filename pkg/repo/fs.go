package repo

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

const (
	repoPermission         = 0755
	defaultRunInfoFilename = "bacalhau.run"
	runInfoFilePermissions = 0755

	ConfigFileName = "config.yaml"

	// UpdateCheckStatePath is the update check paths.
	UpdateCheckStatePath = "update.json"

	// user key paths
	Libp2pPrivateKeyPath = "libp2p_private_key"
	UserPrivateKeyPath   = "user_id.pem"

	// auth paths
	TokensPath = "tokens.json"

	// compute paths
	ComputeStoragesPath = "executor_storages"
	ComputeStorePath    = "compute_store"
	PluginsPath         = "plugins"

	// orchestrator paths
	OrchestratorStorePath = "orchestrator_store"
	AutoCertCachePath     = "autocert-cache"
	NetworkTransportStore = "nats-store"
)

type FsRepoParams struct {
	Path       string
	Migrations *MigrationManager
	Config     *config.Config
}

type FsRepo struct {
	path       string
	Migrations *MigrationManager
	config     *config.Config

	compStrgs   map[string]ComputeStorage
	compStrgsMu sync.Mutex

	exStoreOnce sync.Once
	exStore     store.ExecutionStore
	exStoreErr  error
}

func NewFS(params FsRepoParams) (*FsRepo, error) {
	expandedPath, err := homedir.Expand(params.Path)
	if err != nil {
		return nil, err
	}

	return &FsRepo{
		compStrgs:  make(map[string]ComputeStorage),
		path:       expandedPath,
		Migrations: params.Migrations,
		config:     params.Config,
	}, nil
}

func (fsr *FsRepo) Path() (string, error) {
	if exists, err := fsr.Exists(); err != nil {
		return "", err
	} else if !exists {
		return "", fmt.Errorf("repo is uninitialized")
	}
	return fsr.path, nil
}

func (fsr *FsRepo) Exists() (bool, error) {
	// check if the path is present
	if _, err := os.Stat(fsr.path); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	// check if the repo version file is present
	versionPath := filepath.Join(fsr.path, RepoVersionFile)
	if _, err := os.Stat(versionPath); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	version, err := fsr.readVersion()
	if err != nil {
		return false, err
	}
	if !IsValidVersion(version) {
		return false, NewUnknownRepoVersionError(version)
	}
	return true, nil
}

// Version returns the version of the repo.
func (fsr *FsRepo) Version() (int, error) {
	return fsr.readVersion()
}

func (fsr *FsRepo) Init() error {
	if exists, err := fsr.Exists(); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("cannot init repo: repo already exists")
	}

	log.Info().Msgf("Initializing repo in %q", fsr.path)

	// 0755: Owner can read, write, execute. Others can read and execute.
	if err := os.MkdirAll(fsr.path, repoPermission); err != nil && !os.IsExist(err) {
		return err
	}

	// check if a config file is already present, even though the repo is uninitialized
	// users may still place a config file in a repo (we do this for our terraform deployments)
	// we should attempt to load the config file if it's present.
	if _, err := os.Stat(fsr.join(ConfigFileName)); err == nil {
		if err := fsr.config.Load(fsr.join(ConfigFileName)); err != nil {
			return fmt.Errorf("failed to load config file present in repo: %w", err)
		}
	}

	// ensure all required paths are present in the config, setting any absent ones to the default.
	fsr.ensureRepoPathsConfigured()

	// initialize a node name if one is not present
	if err := fsr.configureNodeName(); err != nil {
		return fmt.Errorf("failed to initalize repo: %w", err)
	}

	if err := fsr.WritePersistedConfigs(); err != nil {
		return fmt.Errorf("failed to persist config: %w", err)
	}

	// initialize the users private key
	if err := fsr.initUserKeys(); err != nil {
		return fmt.Errorf("failed to initialize repo: %w", err)
	}

	// now load the configuration for the repo since we have written it.
	// this ensure commands that create a repo then immediately use the config file
	// have one set in the configuration system
	err := fsr.config.Load(fsr.join(ConfigFileName))
	if err != nil {
		return err
	}

	// TODO this should be a part of the config.
	telemetry.SetupFromEnvs()
	return fsr.writeVersion(RepoVersion3)
}

func (fsr *FsRepo) Open() error {
	// if the repo does not exist we cannot open it.
	if exists, err := fsr.Exists(); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("repo does not exist")
	}

	if fsr.Migrations != nil {
		if err := fsr.Migrations.Migrate(*fsr); err != nil {
			return fmt.Errorf("failed to migrate repo: %w", err)
		}
	}

	// load the configuration for the repo.
	err := fsr.config.Load(fsr.join(ConfigFileName))
	if err != nil {
		return err
	}

	fsr.ensureRepoPathsConfigured()

	cfg, err := fsr.config.Current()
	if err != nil {
		return err
	}

	// ensure the loaded config has valid fields as they pertain to the filesystem
	// e.g. user key and libp2p files exists, storage paths exist, etc.
	if err := validateRepoConfig(cfg); err != nil {
		return fmt.Errorf("failed to validate repo config: %w", err)
	}

	// derive an installationID from the client ID loaded from the repo.
	if cfg.User.InstallationID == "" {
		ID, _ := fsr.GetClientID()
		uuidFromUserID := uuid.NewSHA1(uuid.New(), []byte(ID))
		fsr.config.Set(types.UserInstallationID, uuidFromUserID.String())
	}

	// TODO we should be initializing the file as a part of creating the repo, instead of sometime later.
	if cfg.Update.CheckStatePath == "" {
		cfg.Update.CheckStatePath = filepath.Join(fsr.path, UpdateCheckStatePath)
		fsr.config.Set(types.UpdateCheckStatePath, cfg.Update.CheckStatePath)
	}

	// TODO this should be a part of the config.
	telemetry.SetupFromEnvs()

	return nil
}

// join joins path elements with fsr.path
func (fsr *FsRepo) join(paths ...string) string {
	return filepath.Join(append([]string{fsr.path}, paths...)...)
}

// WritePersistedConfigs will write certain values from the resolved config to the persisted config.
// These include fields for configurations that must not change between version updates, such as the
// execution store and job store paths, in case we change their default values in future updates.
func (fsr *FsRepo) WritePersistedConfigs() error {
	resolvedCfg, err := fsr.config.Current()
	if err != nil {
		return err
	}
	configFilePath := filepath.Join(fsr.path, config.ConfigFileName)
	viperWriter := viper.New()
	viperWriter.SetTypeByDefaultValue(true)
	viperWriter.SetConfigFile(configFilePath)

	// read existing config if it exists.
	if err := viperWriter.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	var fileCfg types.BacalhauConfig
	if err := viperWriter.Unmarshal(&fileCfg, config.DecoderHook); err != nil {
		return err
	}

	// check if any of the values that we want to write are not set in the config file.
	var doWrite bool
	var logMessage strings.Builder
	set := func(key string, value interface{}) {
		viperWriter.Set(key, value)
		logMessage.WriteString(fmt.Sprintf("\n%s:\t%v", key, value))
		doWrite = true
	}
	emptyStoreConfig := types.JobStoreConfig{}
	if fileCfg.Node.Compute.ExecutionStore == emptyStoreConfig {
		set(types.NodeComputeExecutionStore, resolvedCfg.Node.Compute.ExecutionStore)
	}
	if fileCfg.Node.Requester.JobStore == emptyStoreConfig {
		set(types.NodeRequesterJobStore, resolvedCfg.Node.Requester.JobStore)
	}
	if fileCfg.Node.Name == "" && resolvedCfg.Node.Name != "" {
		set(types.NodeName, resolvedCfg.Node.Name)
	}
	if doWrite {
		log.Info().Msgf("Writing to config file %s:%s", configFilePath, logMessage.String())
		return viperWriter.WriteConfig()
	}
	return nil
}

func (fsr *FsRepo) ComputeStoragePath() (string, error) {
	// TODO(forrest) [refactor]: this is just a path in the repo, the config is always going to be under the repo
	// so remove the option to configure it and make it a repo field.
	var cfg types.ComputeStorageConfig
	if err := fsr.config.ForKey(types.NodeComputeStorage, &cfg); err != nil {
		return "", fmt.Errorf("failed to read compute storage configuration: %w", err)
	}
	return cfg.Path, nil
}

func (fsr *FsRepo) ComputeStorage(namespace string) (ComputeStorage, error) {
	fsr.compStrgsMu.Lock()
	defer fsr.compStrgsMu.Unlock()

	rootStoragePath, err := fsr.ComputeStoragePath()
	if err != nil {
		return nil, err
	}

	if cs, ok := fsr.compStrgs[namespace]; ok {
		return cs, nil
	}

	strgPath := fsr.join(rootStoragePath, namespace)
	if err := os.MkdirAll(strgPath, util.OS_USER_RWX); err != nil {
		return nil, err
	}

	cs := &fsCompStrg{
		namespace: namespace,
		path:      strgPath,
	}
	fsr.compStrgs[namespace] = cs

	return cs, nil
}

func (fsr *FsRepo) ExecutionStore(cfg types.JobStoreConfig) (store.ExecutionStore, error) {
	fsr.exStoreOnce.Do(func() {
		// TODO(forrest) [refator] we should base this path on the repo, not the config.
		// Extract the parent directory from the provided path.
		parentDir := filepath.Dir(cfg.Path)

		// Check if the parent directory exists.
		parentInfo, err := os.Stat(parentDir)
		if err != nil {
			if os.IsNotExist(err) {
				// Parent directory does not exist, so create it along with any necessary subdirectories.
				if mkdirErr := os.MkdirAll(parentDir, util.OS_USER_RWX); mkdirErr != nil {
					fsr.exStoreErr = fmt.Errorf("failed to create execution store directory: %s, error: %v", parentDir, mkdirErr)
					return
				}
			} else {
				// Some other error occurred when trying to stat the parent directory.
				fsr.exStoreErr = fmt.Errorf("error checking execution store directory: %s, error: %v", parentDir, err)
				return
			}
		} else if !parentInfo.IsDir() {
			// The parent path exists but is not a directory (e.g., it's a file), return an error.
			fsr.exStoreErr = fmt.Errorf("execution store path was a file, expected a directory: %s", parentDir)
			return
		}

		var exStore store.ExecutionStore
		// TODO(forrest) [refactor] the 'type' of the store should be determined by the repo.
		// The FSRepo can return a store backed by a filesystem, the MemRepo can return
		// a store held in memory (handy for testing)
		switch cfg.Type {
		case types.BoltDB:
			exStore, err = boltdb.NewStore(cfg.Path)
			if err != nil {
				fsr.exStoreErr = err
				return
			}
		default:
			fsr.exStoreErr = fmt.Errorf("unknown JobStore type: %s", cfg.Type)
			return
		}

		fsr.exStore = exStore
	})

	return fsr.exStore, fsr.exStoreErr
}

func (fsr *FsRepo) WriteRunInfo(ctx context.Context, summaryShellVariablesString string) (string, error) {
	runInfoPath := filepath.Join(fsr.path, defaultRunInfoFilename)

	// TODO kill this
	devStackRunInfoPath := os.Getenv("DEVSTACK_ENV_FILE")
	if devStackRunInfoPath != "" {
		runInfoPath = devStackRunInfoPath
	}

	// Use os.Create to truncate the file if it already exists
	f, err := os.Create(runInfoPath)
	if err != nil {
		return "", err
	}
	defer func() {
		err = f.Close()
		if err != nil {
			log.Ctx(ctx).Err(err).Msgf("Failed to close run info file %s", runInfoPath)
		}
	}()

	// Set permissions to constant for read read/write only by user
	err = f.Chmod(runInfoFilePermissions)
	if err != nil {
		return "", err
	}

	_, err = f.Write([]byte(summaryShellVariablesString))
	if err != nil {
		return "", err
	}

	return runInfoPath, nil
}

// loadClientID loads a hash identifying a user based on their ID key.
func (fsr *FsRepo) GetClientID() (string, error) {
	key, err := fsr.loadUserIDKey()
	if err != nil {
		return "", fmt.Errorf("failed to load user ID key: %w", err)
	}

	return convertToClientID(&key.PublicKey), nil
}

const (
	sigHash = crypto.SHA256 // hash function to use for sign/verify
)

// convertToClientID converts a public key to a client ID:
func convertToClientID(key *rsa.PublicKey) string {
	hash := sigHash.New()
	hash.Write(key.N.Bytes())
	hashBytes := hash.Sum(nil)

	return fmt.Sprintf("%x", hashBytes)
}

func (fsr *FsRepo) GetClientPublicKey() (*rsa.PublicKey, error) {
	privKey, err := fsr.loadUserIDKey()
	if err != nil {
		return nil, err
	}
	return &privKey.PublicKey, nil
}

func (fsr *FsRepo) GetClientPrivateKey() (*rsa.PrivateKey, error) {
	privKey, err := fsr.loadUserIDKey()
	if err != nil {
		return nil, err
	}
	return privKey, nil
}

// loadUserIDKey loads the user ID key from whatever source is configured.
func (fsr *FsRepo) loadUserIDKey() (*rsa.PrivateKey, error) {
	keyFile, found := fsr.config.GetString(types.UserKeyPath)
	if !found {
		return nil, fmt.Errorf("config error: user-id-key not set")
	}

	file, err := os.Open(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open user ID key file: %w", err)
	}
	defer closer.CloseWithLogOnError("user ID key file", file)

	keyBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read user ID key file: %w", err)
	}

	keyBlock, _ := pem.Decode(keyBytes)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode user ID key file %q", keyFile)
	}

	// TODO: #3159 Add support for both rsa _and_ ecdsa private keys, see crypto.PrivateKey.
	//       Since we have access to the private key we can hack it by signing a
	//       message twice and comparing them, rather than verifying directly.
	// ecdsaKey, err = x509.ParseECPrivateKey(keyBlock.Bytes)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to parse user: %w", err)
	// }

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user ID key file: %w", err)
	}

	return key, nil
}

// modifies the config to include keys for accessing repo paths
func (fsr *FsRepo) ensureRepoPathsConfigured() {
	fsr.config.SetIfAbsent(types.AuthTokensPath, fsr.join(TokensPath))
	fsr.config.SetIfAbsent(types.UserKeyPath, fsr.join(UserPrivateKeyPath))
	fsr.config.SetIfAbsent(types.NodeExecutorPluginPath, fsr.join(PluginsPath))
	fsr.config.SetIfAbsent(types.NodeComputeStoragePath, fsr.join(ComputeStorePath))
	fsr.config.SetIfAbsent(types.UpdateCheckStatePath, fsr.join(UpdateCheckStatePath))
	fsr.config.SetIfAbsent(types.NodeServerAutoCertCache, fsr.join(AutoCertCachePath))
	fsr.config.SetIfAbsent(types.UserLibp2pKeyPath, fsr.join(Libp2pPrivateKeyPath))
	fsr.config.SetIfAbsent(types.NodeNetworkStoreDir, fsr.join(OrchestratorStorePath, NetworkTransportStore))

	fsr.config.SetIfAbsent(types.NodeRequesterJobStorePath, fsr.join(OrchestratorStorePath, "jobs.db"))
	fsr.config.SetIfAbsent(types.NodeComputeExecutionStorePath, fsr.join(ComputeStoragesPath, "executions.db"))
}

// initUserKeys initializes all files required for a valid bacalhau repo.
func (fsr *FsRepo) initUserKeys() error {
	if err := initUserIDKey(fsr.config.Get(types.UserKeyPath).(string)); err != nil {
		return fmt.Errorf("failed to create user key: %w", err)
	}

	if err := initLibp2pKey(fsr.config.Get(types.UserLibp2pKeyPath).(string)); err != nil {
		return fmt.Errorf("failed to create libp2p key: %w", err)
	}

	return nil
}

// configureNodeName generates a node name if one is not present in the config
func (fsr *FsRepo) configureNodeName() error {
	var nodeName types.NodeID
	if err := fsr.config.ForKey(types.NodeName, &nodeName); err != nil {
		return err
	}

	if nodeName != "" {
		return nil
	}
	var nameProvider string
	if err := fsr.config.ForKey(types.NodeNameProvider, &nameProvider); err != nil {
		return err
	}

	nodeNameProviders := map[string]idgen.NodeNameProvider{
		"hostname": idgen.HostnameProvider{},
		"aws":      idgen.NewAWSNodeNameProvider(),
		"gcp":      idgen.NewGCPNodeNameProvider(),
		"uuid":     idgen.UUIDNodeNameProvider{},
		"puuid":    idgen.PUUIDNodeNameProvider{},
	}
	nodeNameProvider, ok := nodeNameProviders[nameProvider]
	if !ok {
		return fmt.Errorf(
			"unknown node name provider: %s. Supported providers are: %s", nameProvider, lo.Keys(nodeNameProviders))
	}

	name, err := nodeNameProvider.GenerateNodeName(context.TODO())
	if err != nil {
		return err
	}

	// set the new name in the config, so it can be used and persisted later.
	fsr.config.Set(types.NodeName, name)
	return nil
}
