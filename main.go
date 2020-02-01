package main

import (
	"context"
	"fmt"
	"log"

	"github.com/goodgophers/golsp-sdk/server"
	"github.com/sourcegraph/go-lsp"
)

func main() {
	s := server.NewServer()

	s.OnInitialize(func(ctx context.Context, params lsp.InitializeParams) (result interface{}, err error) {
		fmt.Println(params)

		return nil, nil
	})

	log.Fatalln(s.StartTCP(8080))
}
