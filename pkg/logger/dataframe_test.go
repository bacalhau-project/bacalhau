//go:build unit || !integration

package logger

import (
	"bytes"
	"testing"

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
	s.Require().NoError(err)

	s.Require().Equal(original.Tag, df.Tag)
	s.Require().Equal(original.Size, df.Size)
	s.Require().Equal(original.Data, df.Data)
}
