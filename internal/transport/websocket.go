package transport

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonrpc2"
	wsjsonrpc2 "github.com/sourcegraph/jsonrpc2/websocket"
)

type WebsocketTransport struct {
	Addr    string
	Handler jsonrpc2.Handler
}

func NewWebsocketTransport(addr string) *WebsocketTransport {
	return &WebsocketTransport{Addr: addr}
}

func (t *WebsocketTransport) WithHandler(h jsonrpc2.Handler) {
	t.Handler = h
}

func (t *WebsocketTransport) Listen() error {
	listen := func(addr string) (*net.Listener, error) {
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("Could not bind to address %s: %v", addr, err)
			return nil, err
		}

		return &listener, nil
	}

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

		<-jsonrpc2.NewConn(
			context.Background(),
			wsjsonrpc2.NewObjectStream(connection),
			t.Handler).DisconnectNotify()

		log.Printf("langserver-go: connection #%d closed\n", connectionID)
	})

	l, err := listen(t.Addr)
	if err != nil {
		log.Println(err)
		return err
	}
	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  75 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
	log.Println("langserver-go: listening for WebSocket connections on", t.Addr)
	err = server.Serve(*l)
	log.Println(errors.Wrap(err, "HTTP transport"))
	return err
}
