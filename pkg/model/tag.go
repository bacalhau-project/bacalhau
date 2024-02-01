package model

// We use these types to make it harder to accidentally mix up passing the wrong
// annotations to the wrong argument, e.g. avoid Excluded = []string{"included"}
type (
	IncludedTag string
	ExcludedTag string
)

// Set of annotations that will not do any filtering of jobs.
var (
	IncludeAny  = make([]string, 0)
	ExcludeNone = make([]string, 0)
)
