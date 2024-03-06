package boltjobstore

import (
	"fmt"
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/mapping"
	"github.com/pkg/errors"
)

type StoreSearch struct {
	index bleve.Index
}

func NewStoreSearch(jobIndexPath string) (*StoreSearch, error) {
	var err error
	var index bleve.Index

	if index, err = getIndex(jobIndexPath); err != nil {
		return nil, errors.Wrapf(err, "failed to open/create search index at %s", jobIndexPath)
	}

	return &StoreSearch{
		index: index,
	}, nil
}

func (s *StoreSearch) IndexJob(job models.Job) error {
	return s.index.Index(job.ID, job)
}

func (s *StoreSearch) RemoveJob(jobID string) error {
	return s.index.Delete(jobID)
}

// StringForLabel returns a string that can be used to search for a namespace
func (s *StoreSearch) StringForNamespace(ns string) string {
	return fmt.Sprintf("Namespace:%s", ns)
}

// StringForLabel returns a string that can be used to search for a label
// where the value must exist, but we don't care what it is.'
func (s *StoreSearch) StringForLabel(key string) string {
	return fmt.Sprintf("Labels.%s:*", key)
}

// StringForLabel returns a string that can be used to search for a label
// where the value must exist and we care for an exact match
func (s *StoreSearch) StringForLabelValue(key string, val string) string {
	return fmt.Sprintf("Labels.%s:%s", key, val)
}

func (s *StoreSearch) Search(term string, offset int, limit int) ([]string, uint64, error) {
	query := bleve.NewQueryStringQuery(term)
	search := bleve.NewSearchRequest(query)
	search.From = offset
	search.Size = limit
	search.SortBy([]string{"CreateTime"})

	searchResults, err := s.index.Search(search)
	if err != nil {
		return nil, 0, err
	}

	res := make([]string, 0, len(searchResults.Hits))
	for _, hit := range searchResults.Hits {
		res = append(res, hit.ID)
	}
	return res, searchResults.Total, nil
}

func getIndex(path string) (bleve.Index, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		mapping := getMapping()
		index, err := bleve.New(path, mapping)
		return index, err
	}

	return bleve.Open(path)
}

func getMapping() mapping.IndexMapping {
	mapping := bleve.NewIndexMapping()

	jobMapping := bleve.NewDocumentMapping()

	// Enables us to search for "Labels.key:value"
	jobMapping.AddSubDocumentMapping("Labels", bleve.NewDocumentMapping())

	mapping.AddDocumentMapping("Job", jobMapping)
	return mapping
}
