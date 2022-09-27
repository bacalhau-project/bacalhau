package sync

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"

	"github.com/hashicorp/go-multierror"
	"github.com/testground/testground/pkg/logging"
	"nhooyr.io/websocket"
)

var log = logging.S()

type Server struct {
	service Service
	server  *http.Server
	l       net.Listener
}

func NewServer(service Service, port int) (srv *Server, err error) {
	srv = &Server{
		service: service,
	}

	srv.server = &http.Server{
		Handler: http.HandlerFunc(srv.handler),
	}

	srv.l, err = net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return nil, err
	}

	return srv, err
}

func (s *Server) Serve() error {
	return s.server.Serve(s.l)
}

func (s *Server) Addr() string {
	return s.l.Addr().String()
}

func (s *Server) Port() int {
	return s.l.Addr().(*net.TCPAddr).Port
}

func (s *Server) Shutdown(ctx context.Context) error {
	var result *multierror.Error

	result = multierror.Append(
		result,
		s.server.Shutdown(ctx),
		s.service.Close(),
	)

	return result.ErrorOrNil()
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Accept requests from all domains.
	})
	if err != nil {
		log.Warnf("could not upgrade connection: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	conn := &connection{
		Conn:        c,
		service:     s.service,
		ctx:         ctx,
		responses:   make(chan *Response),
		cancelFuncs: map[string]context.CancelFunc{},
	}

	go func() {
		_ = conn.consumeResponses()
	}()
	err = conn.consumeRequests()

	if err == nil {
		_ = c.Close(websocket.StatusNormalClosure, "")
		return
	}

	if errors.Is(err, context.Canceled) ||
		websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
		websocket.CloseStatus(err) == websocket.StatusGoingAway {
		// Client closed the connection by itself.
		log.Info("client closed connection")
		_ = c.Close(websocket.StatusNormalClosure, "")
		return
	}

	log.Warnf("websocket closed unexpectedly: %v", err)
	_ = c.Close(websocket.StatusInternalError, "")
}
