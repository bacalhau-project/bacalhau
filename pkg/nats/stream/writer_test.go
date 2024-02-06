//go:build unit || !integration

package stream

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
)

const overheadWrittenBytes = 23 // additional bytes to wrap the data with StreamingMsg

type WriterTestSuite struct {
	BaseTestSuite
}

func (suite *WriterTestSuite) TestWrite() {
	// Test data
	data := []byte("test data")

	// Execute
	n, err := suite.writer.Write(data)
	suite.Require().NoError(err)

	// Verify
	response := suite.readResponse()
	suite.Require().NoError(response.Err)
	suite.Require().Equal(data, response.Value, "Expected response to be equal to the data")
	suite.Require().Equal(len(data)+overheadWrittenBytes, n, "Expected written bytes to be equal to the length of the data plus overhead")

	// Verify no more records
	suite.Require().Nil(suite.readResponse(), "Expected no more responses after the first one")
}

func (suite *WriterTestSuite) TestWriteObject() {
	// Test object
	obj := map[string]string{"key": "value"}
	data, err := json.Marshal(obj)
	suite.Require().NoError(err)

	// Execute
	n, err := suite.writer.WriteObject(obj)
	suite.Require().NoError(err)

	// Verify
	response := suite.readResponse()
	suite.Require().NoError(response.Err)
	suite.Require().Equal(data, response.Value, "Expected response to be equal to the data")
	suite.Require().Greater(n, 0, "Expected non-zero bytes written")

	// Verify no more records
	suite.Require().Nil(suite.readResponse(), "Expected no more responses after the first one")
}

func (suite *WriterTestSuite) TestClose() {
	// Execute
	err := suite.writer.Close()
	suite.Require().NoError(err)

	// Verify
	response := suite.readResponse()
	suite.Require().Nil(response, "Expected no response after normally closing the writer")
}

func (suite *WriterTestSuite) TestCloseWithCode() {
	// Execute
	closeError := &CloseError{Code: CloseInternalServerErr, Text: "test close error message"}
	err := suite.writer.CloseWithCode(closeError.Code, closeError.Text)
	suite.Require().NoError(err)

	// Verify
	response := suite.readResponse()
	suite.Require().Error(response.Err)
	suite.Require().Equal(closeError, response.Err, "Expected response to be equal to the close error")
	suite.Require().Nil(response.Value, "Expected no response after closing the writer with an error")
}

// Entry point for the test suite
func TestWriterTestSuite(t *testing.T) {
	suite.Run(t, new(WriterTestSuite))
}
