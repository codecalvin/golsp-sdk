package main

import (
	"flag"
	"log"

	"github.com/goodgophers/golsp-sdk/server"
	"github.com/goodgophers/golsp-sdk/transport"
)

var (
	mode = flag.String("mode", "stdio", "communication mode (stdio|tcp|websocket)")
	addr = flag.String("addr", ":4389", "transport listen address (tcp or websocket modes only)")
)

func main() {
	flag.Parse()
	log.SetFlags(0)

	s := server.NewLSPServer()

	switch *mode {
	case "stdio":
		log.Fatal(s.Start(transport.NewStdioTransport()))
	case "tcp":
		log.Fatal(s.Start(transport.NewTCPTransport(*addr)))
	case "websocket":
		log.Fatal(s.Start(transport.NewWebsocketTransport(*addr)))
	}

}
