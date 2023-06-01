package util

import (
	"fmt"
)

type IPLDMap[K comparable, V any] struct {
	Keys   []K
	Values map[K]V
}

func (m *IPLDMap[K, V]) Put(key K, value V) {
	// TODO need to GC keys at some point?
	m.Keys = append(m.Keys, key)
	m.Values[key] = value
}

func (m *IPLDMap[K, V]) Get(key K) V {
	return m.Values[key]
}

func FlattenIPLDMap[K comparable, V any](ipldMap IPLDMap[K, V]) []string {
	var flatMap []string
	for _, key := range ipldMap.Keys {
		value := ipldMap.Values[key]

		// Convert key and value to string
		keyString := fmt.Sprintf("%v", key)
		valueString := fmt.Sprintf("%v", value)

		// Append to flatMap
		flatMap = append(flatMap, keyString)
		flatMap = append(flatMap, valueString)
	}

	return flatMap
}
