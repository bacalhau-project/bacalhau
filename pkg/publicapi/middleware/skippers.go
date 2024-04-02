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

func WebsocketSkipper(c echo.Context) bool {
	return c.Request().Header.Get("Upgrade") == "websocket"
}

// ChainedSkipper creates a skipper that skips if any of the provided skippers returns true
func ChainedSkipper(skippers ...echomiddelware.Skipper) echomiddelware.Skipper {
	return func(c echo.Context) bool {
		for _, skipper := range skippers {
			if skipper(c) {
				return true
			}
		}
		return false
	}
}
