package testapp_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/kyuff/qa/testapp"
)

type testingMMock struct{ code int }

func (m *testingMMock) Run() int { return m.code }

// delayedMock simulates a test run that takes some time, giving subprocesses
// a chance to complete their startup before the stop sequence begins.
type delayedMock struct {
	delay time.Duration
	code  int
}

func (m *delayedMock) Run() int { time.Sleep(m.delay); return m.code }

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
	go srv.Serve(ln)                                         //nolint:errcheck
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
		t.Run("should exit within deadline", func(t *testing.T) {
			// arrange
			var (
				m      = &testingMMock{}
				result = make(chan int, 1)
			)

			// act
			go func() { result <- testapp.Run(m, testapp.WithStub("svc", noopStub())) }()

			// assert
			select {
			case <-result:
			case <-time.After(3 * time.Second):
				t.Fatal("qa.Run did not exit within 3s deadline")
			}
		})

		t.Run("should exit within deadline when app handles SIGINT", func(t *testing.T) {
			// arrange
			var (
				m      = &testingMMock{}
				result = make(chan int, 1)
			)

			// act
			go func() { result <- testapp.Run(m, testapp.WithAppCmd("sleep", "60")) }()

			// assert
			select {
			case <-result:
			case <-time.After(5 * time.Second):
				t.Fatal("qa.Run did not exit within 5s when app handles SIGINT")
			}
		})

		t.Run("should exit within deadline when app ignores SIGINT", func(t *testing.T) {
			// arrange: bash ignores SIGINT via trap; delayedMock holds m.Run long enough
			// for bash to execute the trap before stop() sends SIGINT.
			// qa.Run must fall back to SIGKILL after the 10s stop timeout.
			var (
				m      = &delayedMock{delay: 500 * time.Millisecond}
				result = make(chan int, 1)
			)

			// act
			go func() {
				result <- testapp.Run(m, testapp.WithAppCmd(
					"bash", "-c", "trap '' INT; while true; do sleep 0.1; done",
				))
			}()

			// assert: 500ms startup + 10s kill timeout + buffer
			select {
			case <-result:
			case <-time.After(15 * time.Second):
				t.Fatal("qa.Run did not exit within 15s when app ignores SIGINT")
			}
		})

		t.Run("should start all registered stubs", func(t *testing.T) {
			// arrange
			var (
				stub = noopStub()
				m    = &testingMMock{}
			)

			// act
			testapp.Run(m, testapp.WithStub("svc", stub))

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
			testapp.Run(m, testapp.WithStub("svc", stub))

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
			testapp.Run(m, testapp.WithStub("svc", stub))

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
			testapp.Run(m, testapp.WithStub("svc", stub), testapp.WithControlAddr(cs))

			// assert
			if len(stub.StartCalls()) != 0 {
				t.Errorf("got %d Start calls, want 0", len(stub.StartCalls()))
			}
		})
	})
}
