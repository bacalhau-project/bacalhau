package v1beta2

import "fmt"

// NamespacedID is a tuple of an ID and a namespace
type NamespacedID struct {
	ID        string
	Namespace string
}

// NewNamespacedID returns a new namespaced ID given the ID and namespace
func NewNamespacedID(id, ns string) NamespacedID {
	return NamespacedID{
		ID:        id,
		Namespace: ns,
	}
}

func (n NamespacedID) String() string {
	return fmt.Sprintf("<ns: %q, id: %q>", n.Namespace, n.ID)
}
