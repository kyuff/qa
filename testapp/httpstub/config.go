package httpstub

import "os"

// Option configures an HTTP stub.
type Option func(*config)

// WithAddr sets the default address (host:port) for the stub.
// The URL is derived from this value and set immediately at construction,
// so it is available before Start is called — important when passed to
// WithAppCmd before qa.Run processes its options.
func WithAddr(addr string) Option {
	return func(cfg *config) {
		cfg.addr = addr
	}
}

// WithAddrEnv names an environment variable that overrides WithAddr.
// If the variable is set and non-empty its value is used as the address,
// allowing CI environments (e.g. Docker) to supply a different hostname
// without changing the test code.
//
// Example:
//
//	httpstub.New(
//	    httpstub.WithAddr("localhost:19001"),      // local default
//	    httpstub.WithAddrEnv("PAYMENT_STUB_URL"),  // CI: payments:19001
//	)
func WithAddrEnv(envVar string) Option {
	return func(cfg *config) {
		cfg.addrEnv = envVar
	}
}

type config struct {
	addr    string
	addrEnv string
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

// resolveAddr returns the winning address: env var if set, else the coded default.
func (cfg *config) resolveAddr() string {
	if cfg.addrEnv != "" {
		if v := os.Getenv(cfg.addrEnv); v != "" {
			return v
		}
	}
	return cfg.addr
}
