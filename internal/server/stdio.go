package server

import (
	"context"
	"log"
	"os"

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

type stdrwc struct{}

func (stdrwc) Read(p []byte) (int, error) {
	return os.Stdin.Read(p)
}

func (stdrwc) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

func (stdrwc) Close() error {
	if err := os.Stdin.Close(); err != nil {
		return err
	}
	return os.Stdout.Close()
}
