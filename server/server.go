package server

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
	"github.com/sourcegraph/go-lsp"
)

type Server struct {
	lspCallbacks *jsonrpc.MethodRepository
}

func NewServer() *Server {
	return &Server{lspCallbacks: jsonrpc.NewMethodRepository()}
}

func (s *Server) StartTCP(port int) error {
	log.Printf("[server] starting TCP on port %d...\n", port)
	http.Handle("/", s.lspCallbacks)

	return http.ListenAndServe(fmt.Sprintf(":%d", port), http.DefaultServeMux)
}

func (s *Server) StartStdio() {
	log.Println("[server] starting stdio...")
}

func (s *Server) OnInitialize(do func(ctx context.Context, params lsp.InitializeParams) (result interface{}, err error)) {
	h := Handler{}
	wrapperFunc := func(ctx context.Context, params *fastjson.RawMessage) (result interface{}, rpcErr *jsonrpc.Error) {
		// Unmarshal raw JSON into lsp.InitializeParams
		// Execute user supplied callback
		var initializeParams lsp.InitializeParams

		rpcErr = jsonrpc.Unmarshal(params, &initializeParams)
		if rpcErr != nil {
			log.Println(rpcErr)
			return nil, jsonrpc.ErrInternal()
		}

		res, err := do(ctx, initializeParams)
		if err != nil {
			return nil, jsonrpc.ErrInternal()
		}

		return res, nil
	}
	h.Handle = wrapperFunc

	var result interface{}
	err := s.lspCallbacks.RegisterMethod("initialize", h, lsp.InitializeParams{}, &result)
	if err != nil {
		panic(fmt.Errorf("[server] register OnInitialize: %+v", err))
	}
}

type Handler struct {
	Handle func(c context.Context, params *fastjson.RawMessage) (result interface{}, err *jsonrpc.Error)
}

func (h Handler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (result interface{}, err *jsonrpc.Error) {
	if h.Handle != nil {
		return h.Handle(c, params)
	}

	return nil, nil
}
