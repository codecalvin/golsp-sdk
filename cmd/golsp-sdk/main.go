package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/disposedtrolley/golsp-sdk/langserver"
	"github.com/gorilla/websocket"
	"github.com/sourcegraph/jsonrpc2"
	wsjsonrpc2 "github.com/sourcegraph/jsonrpc2/websocket"
)

var (
	mode         = flag.String("mode", "stdio", "communication mode (stdio|tcp|websocket)")
	addr         = flag.String("addr", ":4389", "server listen address (tcp or websocket)")
	trace        = flag.Bool("trace", false, "print all requests and responses")
	logfile      = flag.String("logfile", "", "also log to this file (in addition to stderr)")
	printVersion = flag.Bool("version", false, "print version and exit")
)

// version is the version field we report back. If you are releasing a new version:
// 1. Create commit without -dev suffix.
// 2. Create commit with version incremented and -dev suffix
// 3. Push to master
// 4. Tag the commit created in (1) with the value of the version string
const version = "v0.1"

func main() {
	flag.Parse()
	log.SetFlags(0)

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	listen := func(addr string) (*net.Listener, error) {
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("Could not bind to address %s: %v", addr, err)
			return nil, err
		}
		if os.Getenv("TLS_CERT") != "" && os.Getenv("TLS_KEY") != "" {
			cert, err := tls.X509KeyPair([]byte(os.Getenv("TLS_CERT")), []byte(os.Getenv("TLS_KEY")))
			if err != nil {
				return nil, err
			}
			listener = tls.NewListener(listener, &tls.Config{
				Certificates: []tls.Certificate{cert},
			})
		}
		return &listener, nil
	}

	if *printVersion {
		fmt.Println(version)
		return nil
	}

	var logW io.Writer
	if *logfile == "" {
		logW = os.Stderr
	} else {
		f, err := os.Create(*logfile)
		if err != nil {
			return err
		}
		defer f.Close()
		logW = io.MultiWriter(os.Stderr, f)
	}
	log.SetOutput(logW)

	var connOpt []jsonrpc2.ConnOpt
	if *trace {
		connOpt = append(connOpt, jsonrpc2.LogMessages(log.New(logW, "", 0)))
	}

	newHandler := func() (jsonrpc2.Handler, io.Closer) {
		return langserver.NewHandler(), ioutil.NopCloser(strings.NewReader(""))
	}

	switch *mode {
	case "tcp":
		lis, err := listen(*addr)
		if err != nil {
			return err
		}
		defer (*lis).Close()

		log.Println("langserver-go: listening for TCP connections on", *addr)

		connectionCount := 0

		for {
			conn, err := (*lis).Accept()
			if err != nil {
				return err
			}
			connectionCount = connectionCount + 1
			connectionID := connectionCount
			log.Printf("langserver-go: received incoming connection #%d\n", connectionID)
			handler, closer := newHandler()
			jsonrpc2Connection := jsonrpc2.NewConn(context.Background(), jsonrpc2.NewBufferedStream(conn, jsonrpc2.VSCodeObjectCodec{}), handler, connOpt...)
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
			handler, closer := newHandler()
			<-jsonrpc2.NewConn(context.Background(), wsjsonrpc2.NewObjectStream(connection), handler, connOpt...).DisconnectNotify()
			err = closer.Close()
			if err != nil {
				log.Println(err)
			}
			log.Printf("langserver-go: connection #%d closed\n", connectionID)
		})

		l, err := listen(*addr)
		if err != nil {
			log.Println(err)
			return err
		}
		server := &http.Server{
			Handler:      mux,
			ReadTimeout:  75 * time.Second,
			WriteTimeout: 60 * time.Second,
		}
		log.Println("langserver-go: listening for WebSocket connections on", *addr)
		err = server.Serve(*l)
		log.Println(errors.Wrap(err, "HTTP server"))
		return err

	case "stdio":
		log.Println("langserver-go: reading on stdin, writing on stdout")
		handler, closer := newHandler()
		<-jsonrpc2.NewConn(context.Background(), jsonrpc2.NewBufferedStream(stdrwc{}, jsonrpc2.VSCodeObjectCodec{}), handler, connOpt...).DisconnectNotify()
		err := closer.Close()
		if err != nil {
			log.Println(err)
		}
		log.Println("connection closed")
		return nil

	default:
		return fmt.Errorf("invalid mode %q", *mode)
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
