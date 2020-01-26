package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/disposedtrolley/golsp-sdk/internal/server"
)

var (
	mode         = flag.String("mode", "stdio", "communication mode (stdio|tcp|websocket)")
	addr         = flag.String("addr", ":4389", "server listen address (tcp or websocket)")
	printVersion = flag.Bool("version", false, "print version and exit")
)

func main() {
	flag.Parse()
	log.SetFlags(0)

	cfg := server.Config{
		Mode:         mode,
		Addr:         addr,
		PrintVersion: printVersion,
	}

	if err := server.Run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
