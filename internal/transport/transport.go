package transport

import "github.com/sourcegraph/jsonrpc2"

type LSPTransport interface {
	Listen() error
	WithHandler(h jsonrpc2.Handler)
}
