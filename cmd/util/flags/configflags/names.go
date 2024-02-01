package configflags

import (
	"fmt"

	"github.com/samber/lo"
)

// FlagDefForKey accepts a configuration key and slice of Definition's containing the key.
// It returns the Definition for the specific key or panics if the key is not part of the definition.
func FlagDefForKey(key string, def ...Definition) Definition {
	f, found := lo.Find(def, func(item Definition) bool {
		return item.ConfigPath == key
	})
	if !found {
		// represents a developer error to call this method with invalid key and definition pair.
		panic(fmt.Sprintf("Key: %s not found in Definition: %+v", key, def))
	}

	return f
}

// FlagNameForKey accepts a configuration key and slice of Definition's containing the key.
// It returns the name of the flag corresponding to the key prefixed with `--`, or panics if the key
// is not part of the definition.
func FlagNameForKey(key string, def ...Definition) string {
	return "--" + FlagDefForKey(key, def...).FlagName
}
