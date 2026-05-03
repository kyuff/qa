package qa

// Config holds the optional configuration for a Runtime or a Run call.
type Config struct {
	stubs []Stub
	app   appController
}

// Option configures a Runtime or a Run call.
type Option func(*Config)

// WithStub registers a Stub with the Runtime. The stub is started during
// NewRuntime (in local and stubs-only modes) and stopped after tests complete.
func WithStub(s Stub) Option {
	return func(cfg *Config) {
		cfg.stubs = append(cfg.stubs, s)
	}
}

// WithApp configures the application to start locally before tests run.
// Pass this to rt.Run, not NewRuntime, so stub URLs are already populated.
// Ignored in stubs-only and ci modes.
func WithApp(cmd string, args ...string) Option {
	return func(cfg *Config) {
		cfg.app = &localApp{cmd: cmd, args: args}
	}
}

func defaultConfig() *Config {
	return applyOptions(&Config{})
}

func applyOptions(cfg *Config, opts ...Option) *Config {
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
