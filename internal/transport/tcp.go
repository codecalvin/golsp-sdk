package transport

import (
	"context"
	"log"
	"net"

	"github.com/sourcegraph/jsonrpc2"
)

type TCPTransport struct {
	Addr    string
	Handler jsonrpc2.Handler
}

func NewTCPTransport(addr string) *TCPTransport {
	return &TCPTransport{Addr: addr}
}

func (t *TCPTransport) WithHandler(h jsonrpc2.Handler) {
	t.Handler = h
}

func (t *TCPTransport) Listen(connOpts []jsonrpc2.ConnOpt) error {
	listen := func(addr string) (*net.Listener, error) {
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("Could not bind to address %s: %v", addr, err)
			return nil, err
		}

		return &listener, nil
	}

	lis, err := listen(t.Addr)
	if err != nil {
		return err
	}
	defer (*lis).Close()

	log.Println("langserver-go: listening for TCP connections on", t.Addr)

	connectionCount := 0

	for {
		conn, err := (*lis).Accept()
		if err != nil {
			return err
		}
		connectionCount = connectionCount + 1
		connectionID := connectionCount
		log.Printf("langserver-go: received incoming connection #%d\n", connectionID)

		jsonrpc2Connection := jsonrpc2.NewConn(
			context.Background(),
			jsonrpc2.NewBufferedStream(conn, jsonrpc2.VSCodeObjectCodec{}),
			t.Handler,
			connOpts...)

		go func() {
			<-jsonrpc2Connection.DisconnectNotify()
			log.Printf("langserver-go: connection #%d closed\n", connectionID)
		}()
	}
}
