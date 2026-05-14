package qa

import (
	"testing"
)

func newCtx[D any](t *testing.T, prefix string, data D, cfg *config) *Ctx[D] {
	return &Ctx[D]{
		T:      t,
		Data:   data,
		prefix: prefix,
		cfg:    cfg,
	}
}

type Ctx[D any] struct {
	T      *testing.T
	Data   D
	prefix string
	cfg    *config
}

// Run will run the test function fn once.
// The fn func can fail t to indicate the test has failed.
func (c *Ctx[D]) Run(name string, fn func(t *testing.T), opts ...Option) bool {
	c.T.Helper()
	return c.T.Run(c.prefix+" "+name, fn)
}

// Eventually calls fn repeatedly until it returns nil or the window expires.
// If fn never returns nil, the step is failed with the last returned error.
func (c *Ctx[D]) Eventually(name string, fn func(t *testing.T) error, opts ...Option) bool {
	c.T.Helper()
	return c.Run(name, func(t *testing.T) {
		t.Helper()
		cfg := applyOptions(c.cfg, opts...)
		var err error
		var attempts uint8
		for attempts = range cfg.windower {
			err = fn(t)
			if err == nil {
				return
			}
		}
		cfg.failFunc(t, "condition not met after %d attempts: %v", attempts, err)
	}, opts...)
}
