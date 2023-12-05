package api

import (
	"context"

	"github.com/labstack/echo/v4"
)

type Server struct {
	echo *echo.Echo
}

func NewServer() (*Server, error) {
	e := echo.New()

	s := &Server{
		echo: e,
	}

	return s, nil
}

func (s *Server) Start() error {
	return s.echo.Start(":22227")
}

func (s *Server) Stop(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}

type Register interface {
	RegisterRoutes(e *echo.Echo)
}

func (s *Server) RegisterAPI(api Register) {
	api.RegisterRoutes(s.echo)
}
