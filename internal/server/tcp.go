package server

import (
	"context"
	"log"
	"net"

	"github.com/disposedtrolley/golsp-sdk/internal/handler"
	"github.com/sourcegraph/jsonrpc2"
)

type TCPServer struct {
	Addr string
}

func NewTCPServer(addr string) *TCPServer {
	return &TCPServer{Addr: addr}
}

func (s *TCPServer) Serve(connOpts []jsonrpc2.ConnOpt) error {
	listen := func(addr string) (*net.Listener, error) {
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("Could not bind to address %s: %v", addr, err)
			return nil, err
		}

		return &listener, nil
	}

	lis, err := listen(s.Addr)
	if err != nil {
		return err
	}
	defer (*lis).Close()

	log.Println("langserver-go: listening for TCP connections on", s.Addr)

	connectionCount := 0

	for {
		conn, err := (*lis).Accept()
		if err != nil {
			return err
		}
		connectionCount = connectionCount + 1
		connectionID := connectionCount
		log.Printf("langserver-go: received incoming connection #%d\n", connectionID)

		h := handler.NewHandler()
		jsonrpc2Connection := jsonrpc2.NewConn(
			context.Background(),
			jsonrpc2.NewBufferedStream(conn, jsonrpc2.VSCodeObjectCodec{}),
			h,
			connOpts...)

		go func() {
			<-jsonrpc2Connection.DisconnectNotify()
			log.Printf("langserver-go: connection #%d closed\n", connectionID)
		}()
	}
}
