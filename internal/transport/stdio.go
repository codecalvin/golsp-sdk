package transport

import (
	"context"
	"log"
	"os"

	"github.com/sourcegraph/jsonrpc2"
)

type StdioTransport struct {
	Handler jsonrpc2.Handler
}

func NewStdioTransport(handler jsonrpc2.Handler) *StdioTransport {
	return &StdioTransport{Handler: handler}
}

func (t *StdioTransport) Listen(connOpts []jsonrpc2.ConnOpt) error {
	log.Println("langserver-go: reading on stdin, writing on stdout")
	<-jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(stdrwc{}, jsonrpc2.VSCodeObjectCodec{}),
		t.Handler,
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
