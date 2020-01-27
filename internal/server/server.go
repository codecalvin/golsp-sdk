package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/go-lsp/lspext"
	"github.com/sourcegraph/jsonrpc2"
)

// NewHandler creates a Go language transport server.
func NewHandler() jsonrpc2.Handler {
	return LSPServer{jsonrpc2.HandlerWithError((&LangHandler{}).Handle)}
}

// lspHandler wraps LangHandler to correctly handle requests in the correct
// order.
//
// The LSP spec dictates a strict ordering that requests should only be
// processed serially in the order they are received. However, implementations
// are allowed to do concurrent computation if it doesn't affect the
// result. We actually can return responses out of order, since vscode does
// not seem to have issues with that. We also do everything concurrently,
// except methods which could mutate the state used by our typecheckers (ie
// textDocument/didOpen, etc). Those are done serially since applying them out
// of order could result in a different textDocument.
type LSPServer struct {
	jsonrpc2.Handler
}

// Handle implements jsonrpc2.Handler
func (h LSPServer) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	// @TODO filesystem ops need to be performed in order.
	go h.Handler.Handle(ctx, conn, req)
}

// LangHandler is a Go language transport LSP/JSON-RPC server.
type LangHandler struct {
	mu       sync.Mutex
	cancel   *cancel
	didInit  bool
	shutdown bool
}

func (h *LangHandler) init() error {
	h.didInit = true
	return nil
}

// reset clears all internal state in h.
func (h *LangHandler) reset() error {
	return nil
}

// Handle creates a response for a JSONRPC2 LSP request. Note: LSP has strict
// ordering requirements, so this should not just be wrapped in an
// jsonrpc2.AsyncHandler. Ensure you have the same ordering as used in the
// NewHandler implementation.
func (h *LangHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	// Prevent any uncaught panics from taking the entire transport down.
	defer func() {
		if r := recover(); r != nil {
			log.Println(err)
			err = r.(error)
		}
	}()

	var cancelManager *cancel
	h.mu.Lock()
	cancelManager = h.cancel
	h.mu.Unlock()

	if req.Method != "initialize" && !h.didInit {
		return nil, errors.New("transport must be initialized")
	}

	if err := h.CheckReady(); err != nil {
		if req.Method == "exit" {
			err = nil
		}
		return nil, err
	}

	// Notifications don't have an ID, so they can't be cancelled
	if cancelManager != nil && !req.Notif {
		// @TODO pass ctx into all server methods
		_, cancel := cancelManager.WithCancel(ctx, req.ID)
		defer cancel()
	}

	switch req.Method {
	case "initialize":
		if h.didInit {
			return nil, errors.New("language transport is already initialized")
		}
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}

		// @TODO unmarshal `initialize` params

		kind := lsp.TDSKFull
		return lsp.InitializeResult{
			Capabilities: lsp.ServerCapabilities{
				TextDocumentSync: &lsp.TextDocumentSyncOptionsOrKind{
					Kind: &kind,
				},
				CompletionProvider:         nil,
				DefinitionProvider:         true,
				TypeDefinitionProvider:     true,
				DocumentFormattingProvider: true,
				DocumentSymbolProvider:     true,
				HoverProvider:              true,
				ReferencesProvider:         true,
				RenameProvider:             true,
			},
		}, nil

	case "initialized":
		// A notification that the client is ready to receive requests. Ignore
		return nil, nil

	case "shutdown":
		h.ShutDown()
		return nil, nil

	case "exit":
		return nil, conn.Close()

	case "$/cancelRequest":
		// notification, don't send back results/errors
		if req.Params == nil {
			return nil, nil
		}

		var params lsp.CancelParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, nil
		}

		if cancelManager != nil {
			cancelManager.Cancel(jsonrpc2.ID{
				Num:      params.ID.Num,
				Str:      params.ID.Str,
				IsString: params.ID.IsString,
			})
		}
		return nil, nil

	case "textDocument/hover":
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		var params lsp.TextDocumentPositionParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return nil, nil

	case "textDocument/definition":
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		var params lsp.TextDocumentPositionParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return nil, nil

	case "textDocument/typeDefinition":
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		var params lsp.TextDocumentPositionParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return nil, nil

	case "textDocument/xdefinition":
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		var params lsp.TextDocumentPositionParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return nil, nil

	case "textDocument/completion":
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		var params lsp.CompletionParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return nil, nil

	case "textDocument/references":
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		var params lsp.ReferenceParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return nil, nil

	case "textDocument/implementation":
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		var params lsp.TextDocumentPositionParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return nil, nil

	case "textDocument/documentSymbol":
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		var params lsp.DocumentSymbolParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return nil, nil

	case "textDocument/signatureHelp":
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		var params lsp.TextDocumentPositionParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return nil, nil

	case "textDocument/formatting":
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		var params lsp.DocumentFormattingParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return nil, nil

	case "workspace/symbol":
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		var params lspext.WorkspaceSymbolParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return nil, nil

	case "workspace/xreferences":
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		var params lspext.WorkspaceReferencesParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return nil, nil

	case "textDocument/rename":
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		var params lsp.RenameParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return nil, nil

	default:
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: fmt.Sprintf("method not supported: %s", req.Method)}
	}
}

func (h *LangHandler) CheckReady() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.shutdown {
		return errors.New("transport is shutting down")
	}
	return nil
}

func (h *LangHandler) ShutDown() {
	h.mu.Lock()
	if h.shutdown {
		log.Printf("Warning: transport received a shutdown request after it was already shut down.")
	}
	h.shutdown = true
	h.mu.Unlock()
}
