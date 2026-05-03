package qa

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// Runtime holds the stubs registered for a test suite.
// Create it in TestMain with NewRuntime and call Run to execute the tests.
type Runtime struct {
	cfg *Config
}

// NewRuntime creates a Runtime and starts all registered stubs.
// In ci mode stubs are not started (they are already running from a
// prior stubs-only invocation at the same configured address).
func NewRuntime(opts ...Option) *Runtime {
	cfg := applyOptions(defaultConfig(), opts...)
	rt := &Runtime{cfg: cfg}
	if currentRunMode() == runModeCI {
		return rt
	}
	for _, s := range cfg.stubs {
		if err := s.Start(context.Background()); err != nil {
			panic(fmt.Sprintf("qa: failed to start stub: %v", err))
		}
	}
	return rt
}

// Run executes the test suite according to the current QA_MODE:
//
//   - local (default): starts the application, runs tests, stops stubs and app.
//   - stubs-only: blocks until all stubs receive a shutdown signal from the
//     ci-mode binary, then exits. Tests are not run.
//   - ci: runs tests, then sends a shutdown signal to all stubs.
//
// Pass WithApp as an option here (not to NewRuntime) so stub URLs are already
// populated when the application command is evaluated.
func (rt *Runtime) Run(m *testing.M, opts ...Option) int {
	runCfg := applyOptions(defaultConfig(), opts...)
	ctx := context.Background()

	switch currentRunMode() {
	case runModeStubsOnly:
		var wg sync.WaitGroup
		for _, s := range rt.cfg.stubs {
			wg.Add(1)
			go func(s Stub) {
				defer wg.Done()
				s.Wait(ctx)
			}(s)
		}
		wg.Wait()
		return 0

	case runModeCI:
		code := m.Run()
		for _, s := range rt.cfg.stubs {
			s.Stop(ctx)
		}
		return code

	default: // local
		if runCfg.app != nil {
			if err := runCfg.app.start(ctx); err != nil {
				panic(fmt.Sprintf("qa: failed to start app: %v", err))
			}
			defer runCfg.app.stop(ctx)
		}
		code := m.Run()
		for _, s := range rt.cfg.stubs {
			s.Stop(ctx)
		}
		return code
	}
}
