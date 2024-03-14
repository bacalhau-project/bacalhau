package repo

import "fmt"

type ErrUnknownRepoVersion int

func NewUnknownRepoVersionError(version int) ErrUnknownRepoVersion {
	return ErrUnknownRepoVersion(version)
}

func (e ErrUnknownRepoVersion) Error() string {
	return fmt.Sprintf("\nUnsupported repository version %d.", e) +
		"\nBacalhau does not know how to read this configuration format and" +
		"\nyou will need to upgrade to a newer version to upgrade this repository."
}
