package server

import (
	"context"

	"github.com/goodgophers/golsp-sdk/internal/transport"
	"github.com/sourcegraph/jsonrpc2"
)

type (
	LSPMethod         string
	LSPMethodCallback func(params interface{}) (result interface{}, err error)
)

type LSPServer struct {
	handler jsonrpc2.Handler
}

func NewLSPServer() LSPServer {
	return LSPServer{handler: jsonrpc2.HandlerWithError(NewLangHandler().Handle)}
}

func (s LSPServer) Start(tsprt transport.LSPTransport) error {
	tsprt.WithHandler(s)
	return tsprt.Listen()
}

// Handle implements jsonrpc2.Handler
func (s LSPServer) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	// @TODO filesystem ops need to be performed in order.
	go s.handler.Handle(ctx, conn, req)
}
