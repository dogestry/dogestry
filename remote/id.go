package remote

import (
	"strings"
)

type ID string

// stripPrefix removes hashing definition that we are not interested in here.
// Also makes sure as to not break backwards compatibility with older versions
// where Docker did not specify the hash function.
func (id ID) trimPrefix() ID {
	return ID(strings.TrimPrefix(string(id), "sha256:"))
}

func (id ID) Short() ID {
	id = id.trimPrefix()
	shortLen := 12
	if len(id) < shortLen {
		shortLen = len(id)
	}
	return ID(id[:shortLen])
}

func (id ID) String() string {
	return string(id.trimPrefix())
}
