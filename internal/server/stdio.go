package server

import (
	"context"
	"log"

	"github.com/disposedtrolley/golsp-sdk/internal/handler"
	"github.com/sourcegraph/jsonrpc2"
)

type StdioServer struct{}

func NewStdioServer() *StdioServer {
	return &StdioServer{}
}

func (s *StdioServer) Serve(connOpts []jsonrpc2.ConnOpt) error {
	h := handler.NewHandler()
	log.Println("langserver-go: reading on stdin, writing on stdout")
	<-jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(stdrwc{}, jsonrpc2.VSCodeObjectCodec{}),
		h,
		connOpts...).DisconnectNotify()

	log.Println("connection closed")
	return nil
}
