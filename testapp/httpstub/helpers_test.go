package httpstub_test

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kyuff/qa/testapp/httpstub"
)

// startedHTTP starts a local HTTP stub on a random port, waits until it is
// ready to serve, wires it through a minimal control server, and registers cleanup.
func startedHTTP(t *testing.T) *httpstub.HTTP {
	t.Helper()
	s := httpstub.New()
	errc := make(chan error, 1)
	go func() { errc <- s.Start(context.Background()) }()

	deadline := time.After(time.Second)
	for {
		select {
		case err := <-errc:
			t.Fatalf("startedHTTP: Start failed: %v", err)
		case <-deadline:
			t.Fatal("startedHTTP: timed out waiting for stub to be ready")
		default:
		}
		if s.Probe() == nil {
			break
		}
		time.Sleep(time.Millisecond)
	}

	// Wire the stub's Handler through a minimal control server so On/Calls work.
	s.SetManagementURL(mountHandler(t, "test", s.Handler()))

	t.Cleanup(func() { s.Stop(context.Background()) })
	return s
}

// mountHandler spins up an HTTP server that mounts h at /stubs/{name}/ and
// returns the management base URL (e.g. http://127.0.0.1:XXXXX/stubs/test).
func mountHandler(t *testing.T, name string, h http.Handler) string {
	t.Helper()
	prefix := "/stubs/" + name
	mux := http.NewServeMux()
	mux.Handle(prefix+"/", http.StripPrefix(prefix, h))

	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("mountHandler: %v", err)
	}
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)                                         //nolint:errcheck
	t.Cleanup(func() { srv.Shutdown(context.Background()) }) //nolint:errcheck

	return "http://" + ln.Addr().String() + prefix
}

func post(t *testing.T, url, body string) int {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("post %s: %v", url, err)
	}
	resp.Body.Close()
	return resp.StatusCode
}
