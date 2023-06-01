package commands

import (
	"encoding/json"

	"golang.org/x/exp/slices"
)

type Command struct {
	Prefix string
	Key    string
	Modify ModifyFunction
}

// ModifyFunction is intended to take data retrieved from the database
// and modify it in some way, returning the newly modified data as a
// []byte. This returned []byte will then be written over the old data.
type ModifyFunction func(existingData []byte) ([]byte, error)

func NewCommand(prefix string, key string, modifyFunc ModifyFunction) Command {
	return Command{
		Prefix: prefix,
		Key:    key,
		Modify: modifyFunc,
	}
}

/*
* Useful, and generic modify functions
 */

// AddToSet is a modify function that deserializes the string list
// in the data parameter, adds a new new valuie
func AddToSet(newValue string) ModifyFunction {
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
