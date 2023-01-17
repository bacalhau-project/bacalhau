package ethblock

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
)

// A log file keeps track of local block ranges and files
// Blocks themselves are pre-processed into csv files
// transactions and receipts stored separately
// do blocks first, then transactions and receipts

// you can attach to an in-process node... in-memory, full-duplex communication

type database struct {
	blockRanges []*model.BlockRange
	blocks string
	log string // serialized block ranges
}

func (db *database) diff(other *model.BlockRange) *model.BlockRange {
	diff := other.Copy()
	for _, blockRange := range db.blockRanges {
		if blockRange.Start.Cmp(diff.End) == -1 {
			diff.End.Set(blockRange.Start)
		}
		if blockRange.End.Cmp(diff.Start) == 1 {
			diff.Start.Set(blockRange.End)
		}
		if diff.Equal(blockRange) {
			return nil
		}
	}
	return diff
}

func (db *database) updateBlockRanges(newBlocks *model.BlockRange) error {
	db.blockRanges = append(db.blockRanges, newBlocks)
	blockRangeBytes, err := json.Marshal(db.blockRanges)
	if err != nil {
		return err
	}
	return os.WriteFile(db.log, blockRangeBytes, 0o066)
}

func (db *database) insert(newBlocks []*types.Block, blockRange *model.BlockRange) error {
	blocksBytes, err := os.ReadFile(db.blocks)
	if err != nil {
		return err
	}
	var blocks []*types.Block
	if err := json.Unmarshal(blocksBytes, &blocks); err != nil {
		return err
	}
	blocks = append(blocks, newBlocks...)
	blocksBytes, err = json.Marshal(blocks)
	if err != nil {
		return err
	}
	// TODO: file mode
	if err := os.WriteFile(db.blocks, blocksBytes, 0o066); err != nil {
		return err
	}

	return db.updateBlockRanges(blockRange)
}

func newDatabase(root string) (*database, error) {
	blocks := filepath.Join(root, "blocks.json")
	if _, err := os.Stat(blocks); os.IsNotExist(err) {
		if _, err := os.Create(blocks); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	blockRanges := make([]*model.BlockRange, 0)
	log := filepath.Join(root, "log.json")
	if _, err := os.Stat(log); os.IsNotExist(err) {
		if _, err := os.Create(log); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		var logBytes []byte
		f, err := os.Open(log)
		if err != nil {
			return nil, err
		}
		if _, err := f.Read(logBytes); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(logBytes, &blockRanges); err != nil {
			return nil, err
		}
	}

	return &database{
		blocks: blocks,
		log: log,
		blockRanges: blockRanges,
	}, nil
}

var bigOne = new(big.Int).SetInt64(1)

type StorageProvider struct {
	db          *database
	fetcher     EthFetcher
	blockRanges []*model.BlockRange
}

type EthFetcher interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
}

func NewStorage(f EthFetcher, localDir string) (*StorageProvider, error) {
	db, err := newDatabase(localDir)
	if err != nil {
		return nil, err
	}
	return &StorageProvider{
		fetcher:     f,
		db: db,
	}, nil
}

func (s *StorageProvider) IsInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

func (s *StorageProvider) HasStorageLocally(ctx context.Context, spec model.StorageSpec) (bool, error) {
	return s.db.diff(spec.BlockRange) == nil, nil
}

func (s *StorageProvider) GetVolumeSize(ctx context.Context, spec model.StorageSpec) (uint64, error) {
	return 0, nil // TODO: how to estimate size of blocks?
}

func (s *StorageProvider) PrepareStorage(ctx context.Context, spec model.StorageSpec) (storage.StorageVolume, error) {
	// TODO PrepareStorage should check if we have the storage locally before fetching it
	diff := s.db.diff(spec.BlockRange)
	if diff != nil {
		blocks := make([]*types.Block, 0)
		for i := new(big.Int).Set(diff.Start); i.Cmp(diff.End) == -1; i.Add(i, bigOne) {
			block, err := s.fetcher.BlockByNumber(ctx, i)
			if err != nil {
				return storage.StorageVolume{}, err
			}
			blocks = append(blocks, block)
		}
		s.db.insert(blocks, diff)
	}
	return 	storage.StorageVolume{
		Type:   storage.StorageVolumeConnectorBind,
		Source: s.db.blocks,
	}, nil
}

func (s *StorageProvider) CleanupStorage(ctx context.Context, spec model.StorageSpec, volume storage.StorageVolume) error {
	pathToCleanup := filepath.Dir(volume.Source)
	return os.RemoveAll(pathToCleanup)
}

// Upload is a no-op because it doesn't make sense for us to "upload" blocks to Ethereum.
func (s *StorageProvider) Upload(ctx context.Context, path string) (model.StorageSpec, error) {
	return model.StorageSpec{}, nil
}

func (s *StorageProvider) Explode(ctx context.Context, spec model.StorageSpec) ([]model.StorageSpec, error) {
	return []model.StorageSpec{spec}, nil
}
