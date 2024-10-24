//go:build unit || !integration

package apimodels

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
)

type APIErrorTestSuite struct {
	suite.Suite
}

func TestAPIErrorSuite(t *testing.T) {
	suite.Run(t, new(APIErrorTestSuite))
}

func (suite *APIErrorTestSuite) TestNewAPIError() {
	tests := []struct {
		name       string
		statusCode int
		message    string
		wantErr    *APIError
	}{
		{
			name:       "basic error",
			statusCode: http.StatusBadRequest,
			message:    "invalid request",
			wantErr: &APIError{
				HTTPStatusCode: http.StatusBadRequest,
				Message:        "invalid request",
				Details:        make(map[string]string),
			},
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			message:    "internal server error",
			wantErr: &APIError{
				HTTPStatusCode: http.StatusInternalServerError,
				Message:        "internal server error",
				Details:        make(map[string]string),
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := NewAPIError(tt.statusCode, tt.message)
			suite.Equal(tt.wantErr.HTTPStatusCode, err.HTTPStatusCode)
			suite.Equal(tt.wantErr.Message, err.Message)
			suite.NotNil(err.Details)
			suite.Empty(err.Details)
		})
	}
}

func (suite *APIErrorTestSuite) TestNewUnauthorizedError() {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "basic unauthorized",
			message: "unauthorized access",
		},
		{
			name:    "empty message",
			message: "",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := NewUnauthorizedError(tt.message)
			suite.Equal(http.StatusUnauthorized, err.HTTPStatusCode)
			suite.Equal(tt.message, err.Message)
			suite.NotNil(err.Details)
		})
	}
}

func (suite *APIErrorTestSuite) TestGenerateAPIErrorFromHTTPResponse() {
	tests := []struct {
		name          string
		response      *http.Response
		responseBody  interface{}
		expectedError *APIError
	}{
		{
			name: "valid error response",
			responseBody: &APIError{
				HTTPStatusCode: http.StatusBadRequest,
				Message:        "invalid input",
				RequestID:      "test-request-id",
				Code:           "BAD_REQUEST",
				Component:      "TestComponent",
				Hint:           "Please check input format",
				Details:        map[string]string{"key": "value"},
			},
		},
		{
			name:     "nil response",
			response: nil,
			expectedError: &APIError{
				HTTPStatusCode: 0,
				Message:        "API call error, invalid response",
				Details:        make(map[string]string),
			},
		},
		{
			name:         "invalid json response",
			responseBody: "invalid json",
			expectedError: &APIError{
				HTTPStatusCode: http.StatusBadRequest,
				Message:        `Unable to parse API call response body. Error: "invalid character 'i' looking for beginning of value". Body received: "invalid json"`,
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			var resp *http.Response
			if tt.response == nil && tt.responseBody != nil {
				var body []byte
				var err error

				switch v := tt.responseBody.(type) {
				case string:
					body = []byte(v)
				default:
					body, err = json.Marshal(tt.responseBody)
					suite.Require().NoError(err)
				}

				resp = &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}
			} else {
				resp = tt.response
			}

			result := GenerateAPIErrorFromHTTPResponse(resp)

			if tt.expectedError != nil {
				suite.Equal(tt.expectedError.HTTPStatusCode, result.HTTPStatusCode)
				suite.Equal(tt.expectedError.Message, result.Message)
			} else {
				originalError := tt.responseBody.(*APIError)
				suite.Equal(originalError.HTTPStatusCode, result.HTTPStatusCode)
				suite.Equal(originalError.Message, result.Message)
				suite.Equal(originalError.RequestID, result.RequestID)
				suite.Equal(originalError.Code, result.Code)
				suite.Equal(originalError.Component, result.Component)
				suite.Equal(originalError.Hint, result.Hint)
				suite.Equal(originalError.Details, result.Details)
			}
		})
	}
}

func (suite *APIErrorTestSuite) TestBacErrorConversion() {
	// Test FromBacError
	suite.Run("FromBacError", func() {
		testCases := []struct {
			name   string
			bacErr bacerrors.Error
			check  func(*APIError)
		}{
			{
				name: "full error conversion",
				bacErr: bacerrors.New("test error").
					WithHTTPStatusCode(http.StatusBadRequest).
					WithCode("TEST_ERROR").
					WithComponent("test-component").
					WithHint("test hint").
					WithDetails(map[string]string{"key": "value"}),
				check: func(apiErr *APIError) {
					suite.Equal(http.StatusBadRequest, apiErr.HTTPStatusCode)
					suite.Equal("test error", apiErr.Message)
					suite.Equal("TEST_ERROR", apiErr.Code)
					suite.Equal("test-component", apiErr.Component)
					suite.Equal("test hint", apiErr.Hint)
					suite.Equal(map[string]string{"key": "value"}, apiErr.Details)
				},
			},
			{
				name:   "minimal error conversion",
				bacErr: bacerrors.New("minimal error"),
				check: func(apiErr *APIError) {
					suite.Equal(http.StatusInternalServerError, apiErr.HTTPStatusCode)
					suite.Equal("minimal error", apiErr.Message)
					suite.Empty(apiErr.Details)
				},
			},
		}

		for _, tc := range testCases {
			suite.Run(tc.name, func() {
				apiErr := FromBacError(tc.bacErr)
				tc.check(apiErr)
			})
		}
	})

	// Test ToBacError
	suite.Run("ToBacError", func() {
		testCases := []struct {
			name   string
			apiErr *APIError
			check  func(bacerrors.Error)
		}{
			{
				name: "full error conversion",
				apiErr: &APIError{
					HTTPStatusCode: http.StatusBadRequest,
					Message:        "test error",
					Code:           "TEST_ERROR",
					Component:      "test-component",
					Hint:           "test hint",
					RequestID:      "test-request-id",
					Details:        map[string]string{"key": "value"},
				},
				check: func(bacErr bacerrors.Error) {
					suite.Equal(http.StatusBadRequest, bacErr.HTTPStatusCode())
					suite.Equal("test error", bacErr.Error())
					suite.Equal(bacerrors.Code("TEST_ERROR"), bacErr.Code())
					suite.Equal("test-component", bacErr.Component())
					suite.Equal("test hint", bacErr.Hint())
					details := bacErr.Details()
					suite.Equal("test-request-id", details["request_id"])
					suite.Equal("value", details["key"])
				},
			},
			{
				name: "nil details conversion",
				apiErr: &APIError{
					HTTPStatusCode: http.StatusBadRequest,
					Message:        "test error",
					RequestID:      "test-request-id",
				},
				check: func(bacErr bacerrors.Error) {
					suite.Equal("test-request-id", bacErr.Details()["request_id"])
				},
			},
		}

		for _, tc := range testCases {
			suite.Run(tc.name, func() {
				bacErr := tc.apiErr.ToBacError()
				tc.check(bacErr)
			})
		}
	})
}

func (suite *APIErrorTestSuite) TestErrorInterface() {
	tests := []struct {
		name string
		err  *APIError
		want string
	}{
		{
			name: "basic error message",
			err:  NewAPIError(http.StatusBadRequest, "test error"),
			want: "test error",
		},
		{
			name: "empty error message",
			err:  NewAPIError(http.StatusBadRequest, ""),
			want: "",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.Equal(tt.want, tt.err.Error())
		})
	}
}

func (suite *APIErrorTestSuite) TestJSONSerialization() {
	tests := []struct {
		name  string
		err   *APIError
		check func(*APIError, *APIError)
	}{
		{
			name: "full error serialization",
			err: &APIError{
				HTTPStatusCode: http.StatusBadRequest,
				Message:        "test error",
				RequestID:      "test-request-id",
				Code:           "TEST_ERROR",
				Component:      "test-component",
				Hint:           "test hint",
				Details:        map[string]string{"key": "value"},
			},
			check: func(original, decoded *APIError) {
				suite.Equal(original.HTTPStatusCode, decoded.HTTPStatusCode)
				suite.Equal(original.Message, decoded.Message)
				suite.Equal(original.RequestID, decoded.RequestID)
				suite.Equal(original.Code, decoded.Code)
				suite.Equal(original.Component, decoded.Component)
				suite.Equal(original.Hint, decoded.Hint)
				suite.Equal(original.Details, decoded.Details)
			},
		},
		{
			name: "minimal error serialization",
			err: &APIError{
				HTTPStatusCode: http.StatusBadRequest,
				Message:        "test error",
			},
			check: func(original, decoded *APIError) {
				suite.Equal(original.HTTPStatusCode, decoded.HTTPStatusCode)
				suite.Equal(original.Message, decoded.Message)
				suite.Nil(decoded.Details)
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			data, err := json.Marshal(tt.err)
			suite.Require().NoError(err)

			var decoded APIError
			suite.Require().NoError(json.Unmarshal(data, &decoded))

			tt.check(tt.err, &decoded)
		})
	}
}
