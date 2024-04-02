//go:build unit || !integration

package logger

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DataFrameTestSuite struct {
	suite.Suite
}

func TestDataFrameTestSuite(t *testing.T) {
	suite.Run(t, new(DataFrameTestSuite))
}

func (s *DataFrameTestSuite) TestBasic() {

	data := []byte("hello")

	original := NewDataFrameFromData(StdoutStreamTag, data)

	buf := bytes.Buffer{}
	buf.Write(original.ToBytes())

	df, err := NewDataFrameFromReader(&buf)
	require.NoError(s.T(), err)

	require.Equal(s.T(), original.Tag, df.Tag)
	require.Equal(s.T(), original.Size, df.Size)
	require.Equal(s.T(), original.Data, df.Data)
}
