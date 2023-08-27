//go:build unit || !integration

package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ZeroLogFormatterTestSuite struct {
	suite.Suite
	formatter *ZeroLogFormatter
	buf       *bytes.Buffer
}

func (suite *ZeroLogFormatterTestSuite) SetupTest() {
	suite.buf = &bytes.Buffer{}
	logger := zerolog.New(suite.buf)
	suite.formatter = NewZeroLogFormatter(
		WithLogger(&logger),
	)
}

func (suite *ZeroLogFormatterTestSuite) TestWriteWithOnlyErrorsTrue() {
	entry := suite.formatter.NewLogEntry(httptest.NewRequest(http.MethodGet, "/", nil))
	entry.Write(http.StatusOK, 123, nil, time.Second, nil)

	// The log buffer should be empty because onlyErrorStatuses is true by default
	assert.Equal(suite.T(), "", suite.buf.String())
}

func (suite *ZeroLogFormatterTestSuite) TestWriteWithOnlyErrorsFalse() {
	suite.formatter.onlyErrorStatuses = false

	entry := suite.formatter.NewLogEntry(httptest.NewRequest(http.MethodGet, "/", nil))
	entry.Write(http.StatusOK, 123, nil, time.Second, nil)

	assert.NotEqual(suite.T(), "", suite.buf.String())
	assert.True(suite.T(), strings.Contains(suite.buf.String(), `"Method":"GET"`))
	assert.True(suite.T(), strings.Contains(suite.buf.String(), `"URI":"/"`))
}

func (suite *ZeroLogFormatterTestSuite) TestPanic() {
	entry := suite.formatter.NewLogEntry(httptest.NewRequest(http.MethodGet, "/", nil))
	entry.Panic("test panic", []byte("stacktrace"))

	assert.Contains(suite.T(), suite.buf.String(), `test panic`)
}

func TestZeroLogFormatterTestSuite(t *testing.T) {
	suite.Run(t, new(ZeroLogFormatterTestSuite))
}
