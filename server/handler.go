package server

import (
	"context"

	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
)

// handler implements the Handler interface expected by the jsonrpc server lib.
//
// The Server creates a new handler for each LSP callback method to be
// implemented, setting its `Handle` field to a wrapped version of the callback
// func provided by the user.
type handler struct {
	Handle func(c context.Context, params *fastjson.RawMessage) (result interface{}, err *jsonrpc.Error)
}

// newHandler takes a CallbackFunc and wraps it in the Handle method of a new
// instance of handler.
func newHandler(serverContext context.Context, do CallbackFunc) handler {
	wrapperFunc := func(ctx context.Context, params *fastjson.RawMessage) (result interface{}, rpcErr *jsonrpc.Error) {
		res, err := do(serverContext, params)
		if err != nil {
			jsonrpcErr := jsonrpc.ErrInternal()
			jsonrpcErr.Message = err.Error()
			return nil, jsonrpcErr
		}

		return res, nil
	}
	return handler{Handle: wrapperFunc}
}

// ServeJSONRPC satisfies the Handler interface expected from the jsonrpc server
// lib.
func (h handler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (result interface{}, err *jsonrpc.Error) {
	if h.Handle != nil {
		return h.Handle(c, params)
	}

	return nil, nil
}
