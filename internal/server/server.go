package server

import (
	"fmt"
	"log"
	"os"

	"github.com/sourcegraph/jsonrpc2"
)

const version = "v0.1"

type Config struct {
	Mode         *string
	Addr         *string
	PrintVersion *bool
}

type LSPServer interface {
	Serve(connOpts []jsonrpc2.ConnOpt) error
}

func Run(cfg Config) error {
	log.SetOutput(os.Stderr)

	if *cfg.PrintVersion {
		fmt.Println(version)
		return nil
	}

	var connOpt []jsonrpc2.ConnOpt

	switch *cfg.Mode {
	case "tcp":
		server := NewTCPServer(*cfg.Addr)
		return server.Serve(connOpt)
	case "websocket":
		server := NewWebsocketServer(*cfg.Addr)
		return server.Serve(connOpt)
	case "stdio":
		server := NewStdioServer()
		return server.Serve(connOpt)
	default:
		return fmt.Errorf("invalid mode %q", *cfg.Mode)
	}
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
