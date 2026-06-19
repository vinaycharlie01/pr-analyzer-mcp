package valueobject

import (
	"fmt"
	"strings"
)

type RepositoryRef struct {
	owner string
	name  string
}

func NewRepositoryRef(owner, name string) (RepositoryRef, error) {
	owner = strings.TrimSpace(owner)
	name = strings.TrimSpace(name)
	if owner == "" {
		return RepositoryRef{}, fmt.Errorf("owner cannot be empty")
	}
	if name == "" {
		return RepositoryRef{}, fmt.Errorf("name cannot be empty")
	}
	return RepositoryRef{owner: owner, name: name}, nil
}

func ParseRepositoryRef(ref string) (RepositoryRef, error) {
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 {
		return RepositoryRef{}, fmt.Errorf("invalid repository ref %q: expected owner/name", ref)
	}
	return NewRepositoryRef(parts[0], parts[1])
}

func (r RepositoryRef) Owner() string { return r.owner }
func (r RepositoryRef) Name() string  { return r.name }
func (r RepositoryRef) String() string { return r.owner + "/" + r.name }
