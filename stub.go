package qa

import "context"

// Stub is the lifecycle interface Run uses to manage any stub server.
// Implement this to register a custom protocol stub with WithStub.
//
// Start must be non-blocking: start the server in the background and return
// once it is ready to accept connections. It is called in local and stubs-only
// modes; in ci mode stubs are assumed to be already running.
//
// Stop sends a shutdown signal. It is called after tests complete in local and
// ci modes, and is sent by the ci-mode binary over the network in stubs-only
// mode, which causes Wait to unblock.
//
// Wait blocks until the stub receives its shutdown signal or ctx is cancelled.
// Used in stubs-only mode to keep the process alive.
type Stub interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context)
	Wait(ctx context.Context)
}
