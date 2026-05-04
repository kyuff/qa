package qa

import (
	"context"
	"net/http"
)

// Stub is the lifecycle interface Run uses to manage any stub server.
// Implement this to register a custom protocol stub with WithStub.
//
// Start must block until Stop is called. It should bind its port and
// begin accepting connections, returning only when the server shuts down.
// Return a non-nil error if the server cannot start (e.g. port in use).
//
// Stop signals the server to shut down, causing Start to return.
//
// Probe returns nil once the stub is ready to accept connections.
// The runtime calls it repeatedly after launching Start in a goroutine,
// waiting until all stubs are ready before proceeding.
//
// Handler returns an http.Handler for management operations (registering
// rules, querying recorded calls). The runtime mounts it on the control
// server at /_qa/stubs/{name}/ so all management traffic is centralised.
type Stub interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context)
	Probe() error
	Handler() http.Handler
}
