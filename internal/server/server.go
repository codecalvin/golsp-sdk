package server

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/disposedtrolley/golsp-sdk/internal/handler"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonrpc2"
	wsjsonrpc2 "github.com/sourcegraph/jsonrpc2/websocket"
)

const version = "v0.1"

type Config struct {
	Mode         *string
	Addr         *string
	PrintVersion *bool
}

func Run(cfg Config) error {
	log.SetOutput(os.Stderr)

	listen := func(addr string) (*net.Listener, error) {
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("Could not bind to address %s: %v", addr, err)
			return nil, err
		}

		return &listener, nil
	}

	if *cfg.PrintVersion {
		fmt.Println(version)
		return nil
	}

	var connOpt []jsonrpc2.ConnOpt

	newHandler := func() (jsonrpc2.Handler, io.Closer) {
		return handler.NewHandler(), ioutil.NopCloser(strings.NewReader(""))
	}

	switch *cfg.Mode {
	case "tcp":
		lis, err := listen(*cfg.Addr)
		if err != nil {
			return err
		}
		defer (*lis).Close()

		log.Println("langserver-go: listening for TCP connections on", *cfg.Addr)

		connectionCount := 0

		for {
			conn, err := (*lis).Accept()
			if err != nil {
				return err
			}
			connectionCount = connectionCount + 1
			connectionID := connectionCount
			log.Printf("langserver-go: received incoming connection #%d\n", connectionID)

			h, closer := newHandler()
			jsonrpc2Connection := jsonrpc2.NewConn(
				context.Background(),
				jsonrpc2.NewBufferedStream(conn, jsonrpc2.VSCodeObjectCodec{}),
				h,
				connOpt...)

			go func() {
				<-jsonrpc2Connection.DisconnectNotify()
				err := closer.Close()
				if err != nil {
					log.Println(err)
				}
				log.Printf("langserver-go: connection #%d closed\n", connectionID)
			}()
		}

	case "websocket":
		mux := http.NewServeMux()
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

		connectionCount := 0

		mux.HandleFunc("/", func(w http.ResponseWriter, request *http.Request) {
			connection, err := upgrader.Upgrade(w, request, nil)
			if err != nil {
				log.Println("error upgrading HTTP to WebSocket:", err)
				http.Error(w, errors.Wrap(err, "could not upgrade to WebSocket").Error(), http.StatusBadRequest)
				return
			}
			defer connection.Close()
			connectionCount = connectionCount + 1
			connectionID := connectionCount

			log.Printf("langserver-go: received incoming connection #%d\n", connectionID)
			h, closer := newHandler()
			<-jsonrpc2.NewConn(
				context.Background(),
				wsjsonrpc2.NewObjectStream(connection),
				h,
				connOpt...).DisconnectNotify()

			err = closer.Close()
			if err != nil {
				log.Println(err)
			}
			log.Printf("langserver-go: connection #%d closed\n", connectionID)
		})

		l, err := listen(*cfg.Addr)
		if err != nil {
			log.Println(err)
			return err
		}
		server := &http.Server{
			Handler:      mux,
			ReadTimeout:  75 * time.Second,
			WriteTimeout: 60 * time.Second,
		}
		log.Println("langserver-go: listening for WebSocket connections on", *cfg.Addr)
		err = server.Serve(*l)
		log.Println(errors.Wrap(err, "HTTP server"))
		return err

	case "stdio":
		log.Println("langserver-go: reading on stdin, writing on stdout")
		h, closer := newHandler()
		<-jsonrpc2.NewConn(
			context.Background(),
			jsonrpc2.NewBufferedStream(stdrwc{}, jsonrpc2.VSCodeObjectCodec{}),
			h,
			connOpt...).DisconnectNotify()

		err := closer.Close()
		if err != nil {
			log.Println(err)
		}
		log.Println("connection closed")
		return nil

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
