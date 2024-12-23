package envelope

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetadataBasicOperations(t *testing.T) {
	t.Run("creation and basic operations", func(t *testing.T) {
		m := make(Metadata)

		// Test Set and Get
		m.Set("key1", "value1")
		assert.Equal(t, "value1", m.Get("key1"))

		// Test Has
		assert.True(t, m.Has("key1"))
		assert.False(t, m.Has("nonexistent"))

		// Test Get with nonexistent key
		assert.Equal(t, "", m.Get("nonexistent"))
	})
}

func TestMetadataBackwardCompatibility(t *testing.T) {
	t.Run("backward compatibility for message type", func(t *testing.T) {
		m := make(Metadata)

		// Test setting new key reflects in legacy key
		m.Set(KeyMessageType, "TestMessage")
		assert.Equal(t, "TestMessage", m[KeyMessageType])
		assert.Equal(t, "TestMessage", m[LegacyMessageType])

		// Test reading from legacy key
		m = make(Metadata)
		m[LegacyMessageType] = "LegacyMessage"
		assert.Equal(t, "LegacyMessage", m.Get(KeyMessageType))
		assert.True(t, m.Has(KeyMessageType))

		// Test Has after deletion
		delete(m, LegacyMessageType)
		assert.False(t, m.Has(KeyMessageType))
		assert.False(t, m.Has(LegacyMessageType))
	})

	t.Run("backward compatibility for payload encoding", func(t *testing.T) {
		m := make(Metadata)

		// Test setting new key reflects in legacy key
		m.Set(KeyPayloadEncoding, "json")
		assert.Equal(t, "json", m[KeyPayloadEncoding])
		assert.Equal(t, "json", m[LegacyEncoding])

		// Test reading from legacy key
		m = make(Metadata)
		m[LegacyEncoding] = "proto"
		assert.Equal(t, "proto", m.Get(KeyPayloadEncoding))
		assert.True(t, m.Has(KeyPayloadEncoding))
		assert.True(t, m.Has(LegacyEncoding))

		// Test Has after deletion
		delete(m, LegacyEncoding)
		assert.False(t, m.Has(KeyPayloadEncoding))
		assert.False(t, m.Has(LegacyEncoding))
	})
}

func TestMetadataNumericOperations(t *testing.T) {
	t.Run("integer operations", func(t *testing.T) {
		m := make(Metadata)

		// Test SetInt and GetInt
		m.SetInt("int", 42)
		assert.Equal(t, 42, m.GetInt("int"))

		// Test invalid int
		m.Set("invalid", "not a number")
		assert.Equal(t, 0, m.GetInt("invalid"))

		// Test nonexistent key
		assert.Equal(t, 0, m.GetInt("nonexistent"))
	})

	t.Run("int64 operations", func(t *testing.T) {
		m := make(Metadata)

		// Test SetInt64 and GetInt64
		var bigNum int64 = 9223372036854775807 // max int64
		m.SetInt64("int64", bigNum)
		assert.Equal(t, bigNum, m.GetInt64("int64"))

		// Test invalid int64
		m.Set("invalid", "not a number")
		assert.Equal(t, int64(0), m.GetInt64("invalid"))
	})

	t.Run("uint64 operations", func(t *testing.T) {
		m := make(Metadata)

		// Test uint64 operations with a valid positive number
		var uintNum uint64 = 1844674407370955161 // large uint64 value within int64 range
		m.SetInt64("uint64", int64(uintNum))
		assert.Equal(t, uintNum, m.GetUint64("uint64"))

		// Test invalid uint64
		m.Set("invalid", "not a number")
		assert.Equal(t, uint64(0), m.GetUint64("invalid"))
	})
}

func TestMetadataTimeOperations(t *testing.T) {
	t.Run("time operations", func(t *testing.T) {
		m := make(Metadata)

		// Test SetTime and GetTime
		now := time.Now()
		m.SetTime("time", now)
		retrieved := m.GetTime("time")
		assert.Equal(t, now.UnixNano(), retrieved.UnixNano())

		// Test invalid time
		m.Set("invalid", "not a time")
		assert.Equal(t, time.Time{}, m.GetTime("invalid"))

		// Test nonexistent key
		assert.Equal(t, time.Time{}, m.GetTime("nonexistent"))
	})
}

func TestMetadataConversion(t *testing.T) {
	t.Run("map conversions", func(t *testing.T) {
		original := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}

		// Test NewMetadataFromMap
		m1 := NewMetadataFromMap(original)
		assert.Equal(t, original["key1"], m1.Get("key1"))
		assert.Equal(t, original["key2"], m1.Get("key2"))

		// Test NewMetadataFromMapCopy
		m2 := NewMetadataFromMapCopy(original)
		assert.Equal(t, original["key1"], m2.Get("key1"))
		assert.Equal(t, original["key2"], m2.Get("key2"))

		// Verify that changes to original don't affect copy
		original["key1"] = "modified"
		assert.NotEqual(t, original["key1"], m2.Get("key1"))

		// Test ToMap
		mapResult := m2.ToMap()
		assert.Equal(t, "value1", mapResult["key1"])
		assert.Equal(t, "value2", mapResult["key2"])

		// Test ToHeaders
		headers := m2.ToHeaders()
		assert.Equal(t, []string{"value1"}, headers["key1"])
		assert.Equal(t, []string{"value2"}, headers["key2"])
	})

	t.Run("nil map handling", func(t *testing.T) {
		m := NewMetadataFromMap(nil)
		assert.NotNil(t, m)
		assert.Equal(t, 0, len(*m))
	})
}
