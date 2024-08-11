package ncl

// NoopMessageFilter is a no-op message filter
type NoopMessageFilter struct{}

// ShouldFilter always returns false
func (n NoopMessageFilter) ShouldFilter(_ *Metadata) bool {
	return false
}

// compile time check for the NoopMessageFilter interface
var _ MessageFilter = NoopMessageFilter{}
