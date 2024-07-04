//go:build unit || !integration

package models_test

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/stretchr/testify/suite"
)

type PagingTokenTestSuite struct {
	suite.Suite
}

func TestPagingTokenSuite(t *testing.T) {
	suite.Run(t, new(PagingTokenTestSuite))
}

func (s *PagingTokenTestSuite) TestNew() {
	type testcase struct {
		name      string
		params    *models.PagingTokenParams
		token     string
		decoded   string
		expectErr bool
	}
	testcases := []testcase{
		{
			name: "valid unreversed",
			params: &models.PagingTokenParams{
				SortBy:      "created_at",
				SortReverse: false,
				Offset:      0,
				Limit:       10,
			},
			token:     "Y3JlYXRlZF9hdDpOOjEwOjA",
			decoded:   "created_at:N:10:0",
			expectErr: false,
		},
		{
			name: "valid reversed",
			params: &models.PagingTokenParams{
				SortBy:      "created_at",
				SortReverse: true,
				Offset:      0,
				Limit:       10,
			},
			token:     "Y3JlYXRlZF9hdDpZOjEwOjA",
			decoded:   "created_at:Y:10:0",
			expectErr: false,
		},
		{
			name: "valid with offset",
			params: &models.PagingTokenParams{
				SortBy:      "created_at",
				SortReverse: true,
				Offset:      10,
				Limit:       10,
			},
			token:     "Y3JlYXRlZF9hdDpZOjEwOjEw",
			decoded:   "created_at:Y:10:10",
			expectErr: false,
		},
		{
			name:      "invalid token",
			params:    &models.PagingTokenParams{},
			token:     "abc",
			decoded:   "created_at:Y:10:10",
			expectErr: true,
		},
	}

	for i := range testcases {
		if !testcases[i].expectErr {
			s.Run(testcases[i].name, func() {

				token := models.NewPagingToken(testcases[i].params)
				s.Equal(testcases[i].token, token.String())
				s.Equal(testcases[i].decoded, token.RawString())
			})
		}

		s.Run(testcases[i].name, func() {
			token, err := models.NewPagingTokenFromString(testcases[i].token)
			if testcases[i].expectErr {
				s.Error(err)
			} else {
				s.NoError(err)
				s.Equal(testcases[i].decoded, token.RawString())
			}

		})
	}

}
