package langserver

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/sourcegraph/jsonrpc2"
	"github.com/stretchr/testify/require"
)

func startServer(t *testing.T, h jsonrpc2.Handler) (addr string, done func()) {
	bindAddr := ":0"
	l, err := net.Listen("tcp", bindAddr)
	if err != nil {
		t.Fatal("Listen:", err)
	}
	go func() {
		if err := serve(context.Background(), l, h); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			t.Fatal("jsonrpc2.Serve:", err)
		}
	}()
	return l.Addr().String(), func() {
		if err := l.Close(); err != nil {
			t.Fatal("close listener:", err)
		}
	}
}

func serve(ctx context.Context, lis net.Listener, h jsonrpc2.Handler, opt ...jsonrpc2.ConnOpt) error {
	for {
		conn, err := lis.Accept()
		if err != nil {
			return err
		}
		jsonrpc2.NewConn(ctx, jsonrpc2.NewBufferedStream(conn, jsonrpc2.VSCodeObjectCodec{}), h, opt...)
	}
}

func dialServer(t *testing.T, addr string, h ...*jsonrpc2.HandlerWithErrorConfigurer) *jsonrpc2.Conn {
	conn, err := (&net.Dialer{}).Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	handler := jsonrpc2.HandlerWithError(func(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (interface{}, error) {
		// no-op
		return nil, nil
	})
	if len(h) == 1 {
		handler = h[0]
	}

	return jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(conn, jsonrpc2.VSCodeObjectCodec{}),
		handler,
	)
}

func TestInitialize(t *testing.T) {
	tests := []struct {
		Name             string
		RPCMethod        string
		RPCParams        map[string]interface{}
		ExpectedResponse map[string]interface{}
	}{
		{
			"when a valid `initialize` call is made",
			"initialize",
			nil,
			make(map[string]interface{}),
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			ExecuteTestCase(t, func(ctx context.Context, conn *jsonrpc2.Conn, notifies chan *jsonrpc2.Request) {
				var result interface{}
				err := conn.Call(ctx, tc.RPCMethod, tc.RPCParams, &result)
				require.Nil(t, err, "should not error on RPC call")
			})
		})
	}
}

func ExecuteTestCase(t *testing.T, fn func(context.Context, *jsonrpc2.Conn, chan *jsonrpc2.Request)) {
	h := NewHandler()

	addr, done := startServer(t, h)
	defer done()

	notifies := make(chan *jsonrpc2.Request, 10)
	conn := dialServer(t, addr, jsonrpc2.HandlerWithError(func(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (interface{}, error) {
		notifies <- req
		return nil, nil
	}))
	defer func() {
		if err := conn.Close(); err != nil {
			t.Fatal("conn.Close:", err)
		}
	}()

	var result interface{}
	ctx := context.Background()
	if err := conn.Call(ctx, "initialize", "", &result); err != nil {
		t.Fatal("initialize:", err)
	}

	fn(ctx, conn, notifies)
}
