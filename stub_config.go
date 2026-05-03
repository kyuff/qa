package qa

// StubConfig holds the optional configuration for an HTTPStub.
type StubConfig struct {
	addr string
}

// StubOption configures an HTTPStub.
type StubOption func(*StubConfig)

// WithAddr sets the host:port the stub server listens on.
// Required when using stubs-only or ci modes, since the application
// must be able to reach the stub at a known address.
// Defaults to "localhost:0" (random port) when omitted.
func WithAddr(addr string) StubOption {
	return func(cfg *StubConfig) {
		cfg.addr = addr
	}
}

func defaultStubConfig() *StubConfig {
	return applyStubOptions(&StubConfig{
		addr: "localhost:0",
	})
}

func applyStubOptions(cfg *StubConfig, opts ...StubOption) *StubConfig {
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
