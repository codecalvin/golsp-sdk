package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"testing"
	"time"

	"github.com/intel-go/fastjson"
	"github.com/stretchr/testify/assert"
	jsonRPCClient "github.com/ybbus/jsonrpc"
)

func TestServer(t *testing.T) {
	tests := []struct {
		Name             string
		RPCMethod        string
		RPCParams        map[string]interface{}
		ExpectedResponse map[string]interface{}
		ExpectedError    *jsonRPCClient.RPCError
	}{
		{
			"when a call to an unsupported method is made",
			"nonexistentMethod",
			make(map[string]interface{}),
			nil,
			&jsonRPCClient.RPCError{
				Code:    -32601,
				Message: "Method not found",
				Data:    nil,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			testPort := getFreePort(t)
			testCtx := context.Background()
			testCtx, cancel := context.WithCancel(testCtx)
			s := NewServer(testCtx)
			go func() {
				s.StartTCP(testPort)
			}()

			time.Sleep(100 * time.Millisecond)

			rpcClient := jsonRPCClient.NewClient(fmt.Sprintf("http://localhost:%d", testPort))

			var result map[string]interface{}
			rpcErr := rpcClient.CallFor(&result, tc.RPCMethod, tc.RPCParams)

			assert.Equal(t, tc.ExpectedResponse, result)
			assert.Equal(t, tc.ExpectedError, rpcErr)

			cancel()
		})
	}
}

func TestRegisterMethod(t *testing.T) {
	tests := []struct {
		Name             string
		RPCMethod        string
		RPCParams        map[string]interface{}
		RPCCallback      CallbackFunc
		ExpectedResponse map[string]interface{}
		ShouldError      bool
		ExpectedError    *jsonRPCClient.RPCError
	}{

		{
			"when a call to a supported method with parameters is made",
			"iAmSupported",
			map[string]interface{}{"paramOne": 1, "paramTwo": 2},
			func(ctx context.Context, params *fastjson.RawMessage) (result interface{}, err error) {
				return params, nil
			},
			map[string]interface{}{"paramOne": float64(1), "paramTwo": float64(2)},
			false,
			nil,
		},
		{
			"when a call to a supported method which should error with parameters is made",
			"iAmSupported",
			map[string]interface{}{"paramOne": 1, "paramTwo": 2},
			func(ctx context.Context, params *fastjson.RawMessage) (result interface{}, err error) {
				return nil, errors.New("test error")
			},
			nil,
			true,
			&jsonRPCClient.RPCError{
				Code:    -32603,
				Message: "test error",
				Data:    nil,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			testPort := getFreePort(t)
			testCtx := context.Background()
			testCtx, cancel := context.WithCancel(testCtx)
			s := NewServer(testCtx)
			s.On(tc.RPCMethod, tc.RPCCallback)
			go func() {
				s.StartTCP(testPort)
			}()

			time.Sleep(100 * time.Millisecond)

			rpcClient := jsonRPCClient.NewClient(fmt.Sprintf("http://localhost:%d", testPort))

			var result map[string]interface{}
			rpcErr := rpcClient.CallFor(&result, tc.RPCMethod, tc.RPCParams)

			assert.EqualValues(t, tc.ExpectedResponse, result)

			if tc.ShouldError {
				assert.Equal(t, tc.ExpectedError, rpcErr)
			}

			cancel()
		})
	}
}

func getFreePort(t *testing.T) int {
	t.Helper()

	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		log.Fatalf("error resolving addr: %+v", err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatalf("error listening on tcp: %+v", err)
	}
	defer func() {
		closeError := l.Close()
		if closeError != nil {
			log.Fatalf("error closing listener: %+v", err)
		}
	}()
	return l.Addr().(*net.TCPAddr).Port
}
