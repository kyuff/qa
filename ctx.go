package qa

import (
	"testing"
	"time"
)

type Ctx[D any] struct {
	T      *testing.T
	Data   D
	Fatalf func(format string, args ...any)
	prefix string
	cfg    *config
}

func (c *Ctx[D]) fail(format string, args ...any) {
	if c.Fatalf != nil {
		c.Fatalf(format, args...)
		return
	}
	c.T.Fatalf(format, args...)
}

func (c *Ctx[D]) Run(name string, fn func(*testing.T), opts ...Option) bool {
	return c.T.Run(c.prefix+" "+name, fn)
}

// Eventually calls fn repeatedly until it returns nil or the window expires.
// If fn never returns nil, the step is failed with the last returned error.
func (c *Ctx[D]) Eventually(name string, window time.Duration, fn func(t *testing.T) error, opts ...Option) {
	c.T.Helper()
	deadline := time.Now().Add(window)
	interval := window / 5
	for {
		err := fn(c.T)
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			c.fail("%s %s: condition not met within %s: %v", c.prefix, name, window, err)
			return
		}
		time.Sleep(interval)
	}
}
