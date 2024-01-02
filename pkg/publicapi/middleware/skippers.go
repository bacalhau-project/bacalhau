package middleware

import (
	"github.com/labstack/echo/v4"
	echomiddelware "github.com/labstack/echo/v4/middleware"
)

func PathMatchSkipper(paths []string) echomiddelware.Skipper {
	skippedPaths := make(map[string]struct{})
	for _, path := range paths {
		skippedPaths[path] = struct{}{}
	}
	return func(c echo.Context) bool {
		_, ok := skippedPaths[c.Path()]
		return ok
	}
}
