package idgen

import (
	"regexp"
)

const ShortIDLength = 8
const ShortIDLengthWithPrefix = ShortIDLength + len(JobIDPrefix)

// Regular expressions for UUID and hostname matching.
var (
	// Regular expression for "prefix-UUID" pattern, where "prefix" is a single character.
	prefixUUIDPattern = regexp.MustCompile(`^([a-zA-Z])-(\w{8})-\w{4}-\w{4}-\w{4}-\w{12}$`)
	// Regular expression for UUID pattern.
	uuidPattern = regexp.MustCompile(`^(\w{8})-\w{4}-\w{4}-\w{4}-\w{12}$`)
	// Regular expression for libp2p peer ID pattern.
	libp2pPattern = regexp.MustCompile(`^Qm[a-zA-Z0-9]{44}$`) // Basic pattern for libp2p peer IDs
)

// ShortUUID takes a string in the format of "prefix-UUID" or just "UUID"
// and returns the prefix along with the first segment of the UUID.
// For example:
// - For "e-78faf114-6a45-457e-825c-40fd2fad768f", it returns "e-78faf114".
// - For "j-78faf114-6a45-457e-825c-40fd2fad768f", it returns "j-78faf114".
// - For "78faf114-6a45-457e-825c-40fd2fad768f", it returns "78faf114".
func ShortUUID(input string) string {
	// Check for "prefix-UUID" pattern.
	if matches := prefixUUIDPattern.FindStringSubmatch(input); matches != nil {
		return matches[1] + "-" + matches[2]
	}

	// Check for UUID pattern.
	if matches := uuidPattern.FindStringSubmatch(input); matches != nil {
		return matches[1]
	}

	// Return input as is for any other format.
	return input
}

// ShortNodeID takes a string in the format of a libp2p peer ID or UUID,
// and returns a shortened version of the input, or the input as is if it doesn't match.
func ShortNodeID(input string) string {
	// Shorten the input if it's a UUID.
	res := ShortUUID(input)
	if len(res) < len(input) {
		return res
	}

	// Check for libp2p peer ID pattern.
	if libp2pPattern.MatchString(input) {
		// Assuming you want the full libp2p ID returned, or you could return a shortened version.
		return input[:ShortIDLength]
	}

	// Return input as is for any other format.
	return input
}
