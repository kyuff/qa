package qa

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// controlServer is the central HTTP management server for a test suite.
// It mounts each stub's Handler at /_qa/stubs/{name}/ and exposes
// POST /_qa/shutdown so the ci-mode binary can stop the stubs-only process.
type controlServer struct {
	addr     string
	URL      string
	mux      *http.ServeMux
	srv      *http.Server
	done     chan struct{}
	doneOnce sync.Once
}

func newControlServer(addr string) *controlServer {
	cs := &controlServer{
		addr: addr,
		mux:  http.NewServeMux(),
		done: make(chan struct{}),
	}
	cs.mux.HandleFunc("POST /_qa/shutdown", cs.handleShutdown)
	return cs
}

func (cs *controlServer) mount(name string, h http.Handler) {
	prefix := "/_qa/stubs/" + name
	cs.mux.Handle(prefix+"/", http.StripPrefix(prefix, h))
}

func (cs *controlServer) start() error {
	ln, err := net.Listen("tcp", cs.addr)
	if err != nil {
		return fmt.Errorf("qa: control server: listen: %w", err)
	}
	cs.URL = "http://" + ln.Addr().String()
	cs.srv = &http.Server{Handler: cs.mux}
	go cs.srv.Serve(ln) //nolint:errcheck
	return nil
}

func (cs *controlServer) stop(ctx context.Context) {
	cs.doneOnce.Do(func() { close(cs.done) })
	if cs.srv != nil {
		cs.srv.Shutdown(ctx) //nolint:errcheck
	}
}

func (cs *controlServer) wait(ctx context.Context) {
	select {
	case <-cs.done:
	case <-ctx.Done():
	}
}

func (cs *controlServer) handleShutdown(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	cs.doneOnce.Do(func() { close(cs.done) })
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cs.srv.Shutdown(ctx) //nolint:errcheck
	}()
}
