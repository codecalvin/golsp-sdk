package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
)

type LangHandler struct {
	mu       sync.Mutex
	cancel   *cancel
	didInit  bool
	shutdown bool
}

func NewLangHandler() *LangHandler {
	return &LangHandler{
		didInit:  false,
		shutdown: false,
	}
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
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}

		if h.didInit {
			return nil, errors.New("language transport is already initialized")
		}

		// @TODO unmarshal `initialize` params
		h.init()

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

	default:
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: fmt.Sprintf("method not supported: %s", req.Method)}
	}
}

func (h *LangHandler) init() {
	h.didInit = true
}

func (h *LangHandler) reset() error {
	return nil
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
