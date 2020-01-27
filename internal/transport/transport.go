package transport

import "github.com/sourcegraph/jsonrpc2"

type LSPTransport interface {
	Listen(connOpts []jsonrpc2.ConnOpt) error
}
