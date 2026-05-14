package qa

type config struct {
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

// Option configures a Run call.
type Option func(*config)
