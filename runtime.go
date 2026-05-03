package qa

import "testing"

// Runtime holds all external dependencies for a test suite.
// Stubs are started when NewRuntime is called, so their addresses are
// available immediately for configuring the application under test.
type Runtime struct {
	stubs []Stub
	app   appController
}

type Option func(*Runtime)

// WithStub registers any Stub implementation with the runtime.
// The stub is started during NewRuntime and stopped after all tests complete.
func WithStub(s Stub) Option {
	return func(rt *Runtime) {
		rt.stubs = append(rt.stubs, s)
		s.Start() // TODO: handle error
	}
}

// WithApp configures the runtime to start the application locally.
// Call this after NewRuntime so that stub addresses are already populated.
// Omit on CI where docker-compose starts the application.
func WithApp(cmd string, args ...string) Option {
	return func(rt *Runtime) {
		rt.app = &localApp{cmd: cmd, args: args}
	}
}

func NewRuntime(opts ...Option) *Runtime {
	rt := &Runtime{}
	for _, opt := range opts {
		opt(rt)
	}
	return rt
}

// Run starts the application (if local), executes all tests, then stops everything.
// Call os.Exit(rt.Run(m)) from TestMain.
func (rt *Runtime) Run(m *testing.M) int {
	if rt.app != nil {
		if err := rt.app.start(); err != nil {
			panic("failed to start app: " + err.Error())
		}
		defer rt.app.stop()
	}
	code := m.Run()
	for _, s := range rt.stubs {
		s.Stop()
	}
	return code
}

type appController interface {
	start() error
	stop()
}

type localApp struct {
	cmd  string
	args []string
}

func (a *localApp) start() error { return nil } // TODO: exec.Command, wait for health check
func (a *localApp) stop()        {}              // TODO: signal, wait
