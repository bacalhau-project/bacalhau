package types

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net/url"
)

type URL struct {
	url.URL
}

// Scan implements sql.Scanner
func (u *URL) Scan(src any) (err error) {
	if src == nil {
		return
	}
	*u, err = ParseURL(fmt.Sprint(src))
	return err
}

// Value implements driver.Valuer
func (u *URL) Value() (driver.Value, error) {
	if u == nil {
		return nil, nil
	} else {
		return u.String(), nil
	}
}

var _ sql.Scanner = (*URL)(nil)
var _ driver.Valuer = (*URL)(nil)

func ParseURL(src string) (u URL, err error) {
	parsed, err := url.Parse(src)
	if parsed != nil {
		u = URL{URL: *parsed}
	}
	return
}

func ParseURLPtr(u *url.URL) *URL {
	if u != nil {
		return &URL{URL: *u}
	}
	return nil
}
