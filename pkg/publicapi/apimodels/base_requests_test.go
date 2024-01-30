//go:build unit || !integration

package apimodels_test

import (
	"encoding/json"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/stretchr/testify/suite"
)

type BaseRequestTestCase struct {
	suite.Suite
}

func TestBaseRequestSuite(t *testing.T) {
	suite.Run(t, new(BaseRequestTestCase))
}

func (s *BaseRequestTestCase) TestBaseRequest() {
	initial := apimodels.BasePutRequest{
		BaseRequest: apimodels.BaseRequest{
			Namespace: "namespace",
			Headers: map[string]string{
				"test": "test",
			},
		},
		IdempotencyToken: "1234",
	}

	b, err := json.Marshal(initial)
	s.Require().NoError(err)

	var result apimodels.BasePutRequest
	err = json.Unmarshal(b, &result)
	s.Require().NoError(err)

	// Should not contain any headers from the TestBaseRequest
	s.Require().Empty(result.Headers, "headers were not excluded from json marshalling")

}
