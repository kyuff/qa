package qa

import (
	"iter"
	"time"

	"github.com/kyuff/qa/internal/seqs"
)

type config struct {
	windower iter.Seq[uint8]
}

func defaultConfig() *config {
	return applyOptions(&config{},
		WithTimeWindow(time.Second, 3),
	)
}

func applyOptions(cfg *config, opts ...Option) *config {
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// Option configures a Run call.
type Option func(*config)

// WithTimeWindow sets the time window for the run.
// A test step will be attempted up to maxRetries times within the given duration.
func WithTimeWindow(dur time.Duration, maxRetries uint8) Option {
	return func(cfg *config) {
		cfg.windower = seqs.NewWindower(dur, maxRetries)
	}
}
