package transport

import (
	"fmt"
	"log"
	"os"

	"github.com/goodgophers/golsp-sdk/internal/server"
	"github.com/sourcegraph/jsonrpc2"
)

const version = "v0.1"

type Config struct {
	Mode         *string
	Addr         *string
	PrintVersion *bool
}

type LSPTransport interface {
	Listen(connOpts []jsonrpc2.ConnOpt) error
}

func Run(cfg Config) error {
	log.SetOutput(os.Stderr)

	if *cfg.PrintVersion {
		fmt.Println(version)
		return nil
	}

	handler := server.NewHandler()
	var connOpt []jsonrpc2.ConnOpt
	switch *cfg.Mode {
	case "tcp":
		transport := NewTCPTransport(handler, *cfg.Addr)
		return transport.Listen(connOpt)
	case "websocket":
		transport := NewWebsocketTransport(handler, *cfg.Addr)
		return transport.Listen(connOpt)
	case "stdio":
		transport := NewStdioTransport(handler)
		return transport.Listen(connOpt)
	default:
		return fmt.Errorf("invalid mode %q", *cfg.Mode)
	}
}
