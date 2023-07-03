package objectstore

import (
	"encoding/json"

	"golang.org/x/exp/slices"
)

type Indexer struct {
	IndexPrefix string
	IndexKey    string
	Operation   IndexUpdateFunction
}

type IndexUpdateFunction func(existingData []byte) ([]byte, error)

func NewIndexer(prefix string, key string, op IndexUpdateFunction) Indexer {
	return Indexer{
		IndexPrefix: prefix,
		IndexKey:    key,
		Operation:   op,
	}
}

/*
* Useful, and generic operations that can be used by indexers to update various
* indices
 */

// AddToSet is a modify function that deserializes the string list
// in the data parameter, adds a new value. The newValue should
// typically be a pointer to another key (to be interpreted by
// the developer).  This can be used for simple values such as
// tags.
func AddToSetOperation(newValue string) IndexUpdateFunction {
	return func(existingData []byte) ([]byte, error) {
		var currentList []string

		if existingData != nil {
			err := json.Unmarshal(existingData, &currentList)
			if err != nil {
				return nil, err
			}
		}

		idx, found := slices.BinarySearch(currentList, newValue)
		if found {
			// Return what we were given as the data already exists
			// in the list
			return existingData, nil
		}

		// Because the binary search above returns the index where the item
		// _would_ be, we can use that to insert into the set and keep it
		// sorted
		currentList = slices.Insert[[]string](currentList, idx, newValue)

		return json.Marshal(&currentList)
	}
}

// DeleteFromSet returns a function. That function will take a json list
// in []byte form and load it before removing `newValue` from the list
// and re-saving it
func DeleteFromSetOperation(valToRemove string) IndexUpdateFunction {
	return func(existingData []byte) ([]byte, error) {
		var currentList []string

		if existingData != nil {
			err := json.Unmarshal(existingData, &currentList)
			if err != nil {
				return nil, err
			}
		} else {
			// No existing data, just return it
			return nil, nil
		}

		idx, found := slices.BinarySearch(currentList, valToRemove)
		if found {
			currentList = slices.Delete(currentList, idx, idx+1)
			return json.Marshal(&currentList)
		}

		// Return what we were given as the data not in the
		// list
		return existingData, nil
	}
}

// AddToMap returns a function that is able to add a pointer to a map
// to reference another type. For example, if a type has a dictionary
// of labels containing things such as Location=X, Production=True then
// these will be stored as a map in a prefix.
func AddToMapOperation(key, value string) IndexUpdateFunction {
	return func(existingData []byte) ([]byte, error) {
		var currentMap map[string][]string

		if existingData != nil {
			err := json.Unmarshal(existingData, &currentMap)
			if err != nil {
				return nil, err
			}
		} else {
			currentMap = make(map[string][]string)
		}

		idx, found := slices.BinarySearch(currentMap[key], value)
		if found {
			return existingData, nil
		}

		currentMap[key] = slices.Insert[[]string](currentMap[key], idx, value)
		return json.Marshal(&currentMap)
	}
}

func DeleteFromMapOperation(key, value string) IndexUpdateFunction {
	return func(existingData []byte) ([]byte, error) {
		var currentMap map[string][]string

		if existingData != nil {
			err := json.Unmarshal(existingData, &currentMap)
			if err != nil {
				return nil, err
			}
		} else {
			return existingData, nil
		}

		// Get the list out of the map item and remove the value
		// from that list
		var items []string
		items, found := currentMap[key]
		if !found {
			return existingData, nil
		}

		idx, found := slices.BinarySearch(items, value)
		if !found {
			return existingData, nil
		}

		currentMap[key] = slices.Delete(items, idx, idx+1)
		return json.Marshal(&currentMap)
	}
}
