package idgen

import "strings"

const ShortIDLength = 8

// ShortID takes a string in the format of "prefix-UUID" or just "UUID"
// and returns the prefix along with the first segment of the UUID.
// For example:
// - For "e-78faf114-6a45-457e-825c-40fd2fad768f", it returns "e-78faf114".
// - For "j-78faf114-6a45-457e-825c-40fd2fad768f", it returns "j-78faf114".
// - For "78faf114-6a45-457e-825c-40fd2fad768f", it returns "78faf114".
func ShortID(input string) string {
	// trying to extract the prefix from the input id
	var prefix, id string
	parts := strings.SplitN(input, "-", 2)
	if len(parts) < 2 {
		id = input  // use the original string if it has less than 2 parts
		prefix = "" // no prefix
	} else if len(parts[0]) > 1 {
		id = input  // use the original string if it has less than 2 parts
		prefix = "" // no prefix
	} else {
		id = parts[1] // use the second part if it is a prefix
		prefix = parts[0]
	}

	// truncate the id to the short id length
	if len(id) > ShortIDLength {
		id = id[:ShortIDLength]
	}

	// append back the prefix if it exists
	if prefix != "" {
		id = prefix + "-" + id
	}
	return id
}
