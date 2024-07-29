//go:build unit || !integration

package ncl

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type SchemaVersionTestSuite struct {
	suite.Suite
}

func (suite *SchemaVersionTestSuite) TestSchemaVersionConstants() {
	suite.Equal(SchemaVersion(1), SchemaVersionJSONV1)
	suite.Equal(SchemaVersion(2), SchemaVersionProtobufV1)
	suite.Equal(SchemaVersionJSONV1, DefaultSchemaVersion)
}

func (suite *SchemaVersionTestSuite) TestSchemaVersionString() {
	testCases := []struct {
		version  SchemaVersion
		expected string
	}{
		{SchemaVersionJSONV1, "json-v1"},
		{SchemaVersionProtobufV1, "protobuf-v1"},
		{SchemaVersion(0), "0x00"},
		{SchemaVersion(255), "0xff"},
	}

	for _, tc := range testCases {
		suite.Run(tc.expected, func() {
			suite.Equal(tc.expected, tc.version.String())
		})
	}
}

func TestSchemaVersionTestSuite(t *testing.T) {
	suite.Run(t, new(SchemaVersionTestSuite))
}
