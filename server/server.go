package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
)

// CallbackFunc defines the function signature that should be implemented to
// process each LSP request.
//
// When processing a request message, the context provided to you may be
// cancelled by the server in response to a subsequent $/cancelRequest
// notification.
//
// It is your responsibility to unmarshal the provided params JSON into the
// correct LSP request types.
//
// You can return either a result (typically a map[string]interface{} JSON
// compatible type, or an error. Errors you return will be wrapped in a
// jsonrpc.ErrInternal type.
type CallbackFunc func(ctx context.Context, params *fastjson.RawMessage) (result interface{}, err error)

// Sever represents an LSP server able to handle connections over TCP or stdio.
type Server struct {
	ctx          context.Context // not used over stdio
	lspCallbacks *jsonrpc.MethodRepository
	httpServer   *http.Server // not used over stdio
}

// NewServer returns a new server using the provided context.
func NewServer(ctx context.Context) *Server {
	return &Server{ctx: ctx, lspCallbacks: jsonrpc.NewMethodRepository()}
}

// StartTCP starts the server in TCP mode, listening to connections on the
// specified port. The server listens until an OS termination signal is received
// or s' context is cancelled.
func (s *Server) StartTCP(port int) {
	httpHandler := http.NewServeMux()
	httpHandler.Handle("/", s.lspCallbacks)
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: httpHandler,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("[server] starting TCP on port %d...\n", port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[server] listen: %+v\n", err)
		}
	}()
	log.Printf("[server] started TCP on port %d\n", port)

	select {
	case <-s.ctx.Done():
		log.Println("[server] stopping - context done")
		s.Stop()
	case <-done:
		log.Println("[server] stopping - OS signal")
		s.Stop()
	}
}

func (s *Server) StartStdio() {
	log.Println("[server] starting stdio...")
}

// Stop gracefully shuts down the server if it was listening over TCP.
func (s *Server) Stop() {
	if s.httpServer != nil {
		log.Printf("[server] stopped")

		if err := s.httpServer.Shutdown(s.ctx); err != nil {
			log.Fatalf("[server] shutdown failed: %+v\n", err)
		}
		log.Println("[server] exited properly")
	}
}

// On registers an LSP callback function for a method. The method should be a
// request or notification method defined in the LSP Specification.
func (s *Server) On(method string, do func(ctx context.Context, params *fastjson.RawMessage) (result interface{}, err error)) {
	h := newHandler(s.ctx, do)

	var result interface{}
	// @TODO not sure what params is used for.
	err := s.lspCallbacks.RegisterMethod(method, h, nil, &result)
	if err != nil {
		panic(fmt.Errorf("[server] register %s: %+v", method, err))
	}
}
