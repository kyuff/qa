package qa

import "os"

type namedStub struct {
	name string
	stub Stub
}

type config struct {
	namedStubs  []namedStub
	app         appController
	controlAddr string
	controlEnv  string
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

// WithApp configures the application to start locally before tests run.
// Ignored in stubs-only and ci modes.
func WithApp(cmd string, args ...string) Option {
	return func(cfg *config) {
		cfg.app = &localApp{cmd: cmd, args: args}
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
