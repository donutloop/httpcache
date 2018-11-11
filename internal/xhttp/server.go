package xhttp

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

type Server struct {
	*http.Server
	Listener net.Listener

	ShutdownTimeout time.Duration

	Logger       *log.Logger
}

// Start starts the server and waits for it to return.
func (s *Server) Start() error {
	s.Logger.Println(fmt.Sprintf("starting server on (%v)", s.Listener.Addr()))

	return s.Serve(s.Listener)
}

// Start starts the server and waits for it to return.
func (s *Server) StartTLS(certFile, keyFile string) error {
	s.Logger.Println(fmt.Sprintf("starting server on (%v)", s.Listener.Addr()))

	return s.ServeTLS(s.Listener, certFile, keyFile)
}

//  srv.ServeTLS(tcpKeepAliveListener{ln.(*net.TCPListener)}, certFile, keyFile)

// Stop tries to shut the server down gracefully first, then forcefully closes it.
func (s *Server) Stop() {
	ctx := context.Background()
	if s.ShutdownTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), s.ShutdownTimeout)

		defer cancel()
	}

	s.Logger.Println("shutting server down")
	err := s.Server.Shutdown(ctx)
	if err != nil {
		s.Logger.Println(err)
	}

	s.Server.Close()
}

