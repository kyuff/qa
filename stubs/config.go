package stubs

// Option configures an HTTP stub.
type Option func(*config)

// WithAddr sets a fixed host:port for the stub server.
// Required when the application under test must reach the stub at a known
// address — for example in stubs-only and ci modes.
func WithAddr(addr string) Option {
	return func(cfg *config) {
		cfg.addr = addr
	}
}

type config struct {
	addr string
}

func defaultConfig() *config {
	return &config{addr: "localhost:0"}
}

func applyOptions(cfg *config, opts ...Option) *config {
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
