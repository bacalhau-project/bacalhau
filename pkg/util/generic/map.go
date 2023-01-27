package generic

// Map transforms a slice into a slice of another type
func Map[F any, T any](inputs []F, f func(F) T) []T {
	tees := make([]T, 0, len(inputs))
	for _, input := range inputs {
		tees = append(tees, f(input))
	}
	return tees
}
