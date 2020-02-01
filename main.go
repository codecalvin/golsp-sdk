package main

import (
	"context"
	"fmt"

	"github.com/goodgophers/golsp-sdk/server"
	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
	"github.com/sourcegraph/go-lsp"
)

func main() {
	ctx := context.Background()
	s := server.NewServer(ctx)

	s.On("initialize", func(ctx context.Context, params *fastjson.RawMessage) (result interface{}, err error) {
		var initializeParams lsp.InitializeParams
		if err := jsonrpc.Unmarshal(params, &initializeParams); err != nil {
			return nil, err
		}
		fmt.Printf("%+v\n", initializeParams)

		return nil, nil
	})

	s.StartTCP(8080)
}
