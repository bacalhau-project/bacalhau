package objectstore

// Indexable defines the interface we expect any entities being stored to
// implement.  This allows us to ask the entities how they would like to
// be indexed (or how to clean up after deletion).
type Indexable interface {
	OnUpdate() []Indexer
	OnDelete() []Indexer
}
