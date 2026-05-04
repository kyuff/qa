package qa

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// testingM is the subset of *testing.M used by Run.
type testingM interface {
	Run() int
}

// managementURLSetter is implemented by stubs that need to know their
// management URL on the control server (e.g. to route On/Calls calls).
type managementURLSetter interface {
	SetManagementURL(url string)
}

// Run executes the test suite according to the current QA_MODE:
//
//   - local (default): starts the control server and all stubs, runs tests, stops everything.
//   - stubs-only: starts the control server and all stubs, then blocks until the
//     ci-mode binary sends POST /_qa/shutdown to the control server.
//   - ci: configures stubs to proxy to the stubs-only control server, runs tests,
//     then sends shutdown to the control server.
func Run(m testingM, opts ...Option) int {
	cfg := applyOptions(defaultConfig(), opts...)
	ctx := context.Background()
	controlAddr := resolveControlAddr(cfg)

	switch currentRunMode() {
	case runModeCI:
		controlURL := "http://" + controlAddr
		setManagementURLs(cfg.namedStubs, controlURL)

		code := m.Run()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, controlURL+"/_qa/shutdown", nil)
		if err == nil {
			if resp, err := http.DefaultClient.Do(req); err == nil {
				resp.Body.Close()
			}
		}
		return code

	default: // local and stubs-only share startup; differ only in what runs next
		cs := newControlServer(controlAddr)
		for _, ns := range cfg.namedStubs {
			cs.mount(ns.name, ns.stub.Handler())
		}
		if err := cs.start(); err != nil {
			panic(fmt.Sprintf("qa: %v", err))
		}

		setManagementURLs(cfg.namedStubs, cs.URL)

		stubCtx, stubCancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup
		for _, ns := range cfg.namedStubs {
			wg.Add(1)
			go func(s Stub) {
				defer wg.Done()
				if err := s.Start(stubCtx); err != nil {
					// port-in-use and similar errors surface here
					fmt.Printf("qa: stub exited: %v\n", err)
				}
			}(ns.stub)
		}

		probeCtx, probeCancel := context.WithTimeout(stubCtx, 30*time.Second)
		probeAll(probeCtx, cfg.namedStubs)
		probeCancel()

		var code int
		if currentRunMode() == runModeStubsOnly {
			cs.wait(ctx)
		} else {
			if cfg.app != nil {
				if err := cfg.app.start(ctx); err != nil {
					panic(fmt.Sprintf("qa: app: %v", err))
				}
				if cfg.appHealthURL != "" {
					healthCtx, healthCancel := context.WithTimeout(ctx, 30*time.Second)
					probeURL(healthCtx, cfg.appHealthURL)
					healthCancel()
				}
			}
			code = m.Run()
		}

		stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer stopCancel()
		if cfg.app != nil {
			cfg.app.stop(stopCtx)
		}
		for _, ns := range cfg.namedStubs {
			ns.stub.Stop(stopCtx)
		}
		stubCancel()
		wg.Wait()
		cs.stop(stopCtx)
		return code
	}
}

func setManagementURLs(stubs []namedStub, controlURL string) {
	for _, ns := range stubs {
		if s, ok := ns.stub.(managementURLSetter); ok {
			s.SetManagementURL(controlURL + "/_qa/stubs/" + ns.name)
		}
	}
}

func probeURL(ctx context.Context, url string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		resp, err := http.Get(url) //nolint:noctx
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func probeAll(ctx context.Context, stubs []namedStub) {
	var wg sync.WaitGroup
	for _, ns := range stubs {
		wg.Add(1)
		go func(s Stub) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				if s.Probe() == nil {
					return
				}
				time.Sleep(5 * time.Millisecond)
			}
		}(ns.stub)
	}
	wg.Wait()
}
