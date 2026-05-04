package qa

import "os"

type namedStub struct {
	name string
	stub Stub
}

type config struct {
	namedStubs   []namedStub
	app          appController
	appHealthURL string
	controlAddr  string
	controlEnv   string
}

// Option configures a Run call.
type Option func(*config)

// WithStub registers a stub under a logical name used for control server routing.
// The name should match the Docker service name in CI environments so that
// hostname-based routing works without additional configuration.
func WithStub(name string, s Stub) Option {
	return func(cfg *config) {
		cfg.namedStubs = append(cfg.namedStubs, namedStub{name: name, stub: s})
	}
}

// WithAppCmd configures a command to start locally before tests run.
// The child process inherits the test process environment.
// Ignored in stubs-only and ci modes.
func WithAppCmd(cmd string, args ...string) Option {
	return func(cfg *config) {
		cfg.app = &localApp{cmd: cmd, args: args}
	}
}

// WithAppHealthCheck configures a URL to probe after starting the app locally.
// qa.Run blocks until the URL returns 200 OK before running tests.
// Uses the same 30-second timeout as stub probing.
// Ignored in stubs-only and ci modes.
func WithAppHealthCheck(url string) Option {
	return func(cfg *config) {
		cfg.appHealthURL = url
	}
}

// WithControlAddr sets the address for the central control server.
// Required in stubs-only and ci modes so both processes agree on the address.
// Defaults to a random port suitable for local-only use.
func WithControlAddr(addr string) Option {
	return func(cfg *config) {
		cfg.controlAddr = addr
	}
}

// WithControlAddrEnv names an environment variable that overrides WithControlAddr.
// Use this to configure the control server address in CI without hardcoding it.
func WithControlAddrEnv(envVar string) Option {
	return func(cfg *config) {
		cfg.controlEnv = envVar
	}
}

func defaultConfig() *config {
	return &config{controlAddr: "localhost:0"}
}

func applyOptions(cfg *config, opts ...Option) *config {
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

func resolveControlAddr(cfg *config) string {
	if cfg.controlEnv != "" {
		if v := os.Getenv(cfg.controlEnv); v != "" {
			return v
		}
	}
	return cfg.controlAddr
}
