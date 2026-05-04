package qa_test

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/kyuff/qa"
)

type testingMMock struct{ code int }

func (m *testingMMock) Run() int { return m.code }

// startFakeControlServer starts a minimal HTTP server that absorbs the
// /_qa/shutdown POST sent by the ci-mode runtime, returning its host:port.
func startFakeControlServer(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("startFakeControlServer: %v", err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})}
	go srv.Serve(ln) //nolint:errcheck
	t.Cleanup(func() { srv.Shutdown(context.Background()) }) //nolint:errcheck
	return ln.Addr().String()
}

func noopStub() *StubMock {
	return &StubMock{
		StartFunc:   func(ctx context.Context) error { <-ctx.Done(); return nil },
		StopFunc:    func(ctx context.Context) {},
		ProbeFunc:   func() error { return nil },
		HandlerFunc: func() http.Handler { return http.NotFoundHandler() },
	}
}

func TestRun(t *testing.T) {
	t.Run("local mode", func(t *testing.T) {
		t.Run("should start all registered stubs", func(t *testing.T) {
			// arrange
			var (
				stub = noopStub()
				m    = &testingMMock{}
			)

			// act
			qa.Run(m, qa.WithStub("svc", stub))

			// assert
			if len(stub.StartCalls()) != 1 {
				t.Errorf("got %d Start calls, want 1", len(stub.StartCalls()))
			}
		})

		t.Run("should probe all stubs before running tests", func(t *testing.T) {
			// arrange
			var (
				stub = noopStub()
				m    = &testingMMock{}
			)

			// act
			qa.Run(m, qa.WithStub("svc", stub))

			// assert
			if len(stub.ProbeCalls()) == 0 {
				t.Error("expected Probe to be called at least once")
			}
		})

		t.Run("should stop all registered stubs after tests", func(t *testing.T) {
			// arrange
			var (
				stub = noopStub()
				m    = &testingMMock{}
			)

			// act
			qa.Run(m, qa.WithStub("svc", stub))

			// assert
			if len(stub.StopCalls()) != 1 {
				t.Errorf("got %d Stop calls, want 1", len(stub.StopCalls()))
			}
		})
	})

	t.Run("ci mode", func(t *testing.T) {
		t.Run("should not start stubs", func(t *testing.T) {
			t.Setenv("QA_MODE", "ci")

			// arrange: start a real control server so the ci shutdown POST has somewhere to go
			var (
				stub = noopStub()
				m    = &testingMMock{}
				cs   = startFakeControlServer(t)
			)

			// act
			qa.Run(m, qa.WithStub("svc", stub), qa.WithControlAddr(cs))

			// assert
			if len(stub.StartCalls()) != 0 {
				t.Errorf("got %d Start calls, want 0", len(stub.StartCalls()))
			}
		})
	})
}
