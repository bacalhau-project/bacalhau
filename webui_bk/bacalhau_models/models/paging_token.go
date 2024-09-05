package models

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const (
	delimiter         = ":"
	expectedPartCount = 4
)

type PagingTokenParams struct {
	SortBy      string
	SortReverse bool
	Limit       uint32
	Offset      uint32
}

type PagingToken struct {
	SortBy      string
	SortReverse bool
	Limit       uint32
	Offset      uint32
}

func NewPagingToken(params *PagingTokenParams) *PagingToken {
	return &PagingToken{
		SortBy:      params.SortBy,
		SortReverse: params.SortReverse,
		Limit:       params.Limit,
		Offset:      params.Offset,
	}
}

func NewPagingTokenFromString(s string) (*PagingToken, error) {
	var err error
	var decodedBytes []byte

	if decodedBytes, err = base64.RawURLEncoding.DecodeString(s); err != nil {
		return nil, NewErrInvalidPagingToken(s, "failed to decode paging token")
	}

	parts := strings.Split(string(decodedBytes), delimiter)
	if len(parts) != expectedPartCount {
		return nil, NewErrInvalidPagingToken(s, "invalid number of parts")
	}

	token := &PagingToken{
		SortBy:      parts[0],
		SortReverse: parts[1] == "Y",
	}

	if limit, err := strconv.ParseUint(parts[2], 10, 32); err != nil {
		return nil, NewErrInvalidPagingToken(s, "malformed token")
	} else {
		token.Limit = uint32(limit)
	}

	if offset, err := strconv.ParseUint(parts[3], 10, 32); err != nil {
		return nil, NewErrInvalidPagingToken(s, "malformed token")
	} else {
		token.Offset = uint32(offset)
	}

	return token, nil
}

func (pagingToken *PagingToken) RawString() string {
	reverse := "N"
	if pagingToken.SortReverse {
		reverse = "Y"
	}

	return strings.Join([]string{
		pagingToken.SortBy,
		reverse,
		strconv.FormatUint(uint64(pagingToken.Limit), 10),
		strconv.FormatUint(uint64(pagingToken.Offset), 10),
	}, delimiter)
}

// String returns the token as a base 64 encoded string where each field is
// delimited.
func (pagingToken *PagingToken) String() string {
	return base64.RawURLEncoding.EncodeToString([]byte(pagingToken.RawString()))
}

func NewErrInvalidPagingToken(s string, msg string) error {
	return errors.Wrap(fmt.Errorf("invalid paging token: %s", s), msg)
}
