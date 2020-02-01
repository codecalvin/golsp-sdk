package server

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	jsonRPCClient "github.com/ybbus/jsonrpc"
)

var port = 8080

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
			go startServer(port)

			rpcClient := jsonRPCClient.NewClient(fmt.Sprintf("http://localhost:%d", port))

			var result map[string]interface{}
			rpcErr := rpcClient.CallFor(&result, tc.RPCMethod, tc.RPCParams)

			assert.Equal(t, tc.ExpectedResponse, result)
			assert.Equal(t, tc.ExpectedError, rpcErr)
		})
	}
}

func startServer(port int) {
	s := NewServer()
	err := s.StartTCP(port)
	if err != nil {
		panic(fmt.Errorf("start server on port %d for testing: %+v", port, err))
	}
}
