package main

import (
	"flag"
	"log"

	"github.com/goodgophers/golsp-sdk/internal/server"
	"github.com/goodgophers/golsp-sdk/internal/transport"
)

var (
	mode         = flag.String("mode", "stdio", "communication mode (stdio|tcp|websocket)")
	addr         = flag.String("addr", ":4389", "transport listen address (tcp or websocket)")
	printVersion = flag.Bool("version", false, "print version and exit")
)

func main() {
	flag.Parse()
	log.SetFlags(0)

	s := server.NewLSPServer()
	s.Start(transport.NewTCPTransport(":4389"))
}
