package qa

import (
	"context"
	"fmt"
	"sync"
)

// testingM is the subset of *testing.M used by Run.
type testingM interface {
	Run() int
}

// Run executes the test suite according to the current QA_MODE:
//
//   - local (default): starts all stubs and the application, runs tests, then stops everything.
//   - stubs-only: starts all stubs and blocks until they receive a shutdown signal
//     from the ci-mode binary. Tests are not run.
//   - ci: runs tests, then sends a shutdown signal to all stubs.
//
// Stubs that an application connects to must use stubs.WithAddr so their URL is
// known at argument-evaluation time, before Run is called.
func Run(m testingM, opts ...Option) int {
	cfg := applyOptions(defaultConfig(), opts...)
	ctx := context.Background()

	if currentRunMode() != runModeCI {
		for _, s := range cfg.stubs {
			if err := s.Start(ctx); err != nil {
				panic(fmt.Sprintf("qa: failed to start stub: %v", err))
			}
		}
	}

	switch currentRunMode() {
	case runModeStubsOnly:
		var wg sync.WaitGroup
		for _, s := range cfg.stubs {
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
		for _, s := range cfg.stubs {
			s.Stop(ctx)
		}
		return code

	default: // local
		if cfg.app != nil {
			if err := cfg.app.start(ctx); err != nil {
				panic(fmt.Sprintf("qa: failed to start app: %v", err))
			}
			defer cfg.app.stop(ctx)
		}
		code := m.Run()
		for _, s := range cfg.stubs {
			s.Stop(ctx)
		}
		return code
	}
}
