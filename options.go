package qa

type config struct {
	stubs []Stub
	app   appController
}

// Option configures a Run call.
type Option func(*config)

// WithStub registers a Stub to be started before tests run and stopped after.
// In ci mode the stub is not started (it is already running from the prior
// stubs-only invocation).
func WithStub(s Stub) Option {
	return func(cfg *config) {
		cfg.stubs = append(cfg.stubs, s)
	}
}

// WithApp configures the application to start locally before tests run.
// Ignored in stubs-only and ci modes.
func WithApp(cmd string, args ...string) Option {
	return func(cfg *config) {
		cfg.app = &localApp{cmd: cmd, args: args}
	}
}

func defaultConfig() *config {
	return applyOptions(&config{})
}

func applyOptions(cfg *config, opts ...Option) *config {
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
