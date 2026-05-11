package qa

import (
	"testing"
	"time"
)

type TestingT interface {
	Helper()
	Fatalf(format string, args ...any)
}

type Ctx[D any] struct {
	T      *testing.T
	Data   D
	prefix string
}

func (c *Ctx[D]) Run(name string, fn func(*testing.T)) bool {
	return c.T.Run(c.prefix+" "+name, fn)
}

// Eventually calls fn repeatedly until it returns nil or the window expires.
// If fn never returns nil, the step is failed with the last returned error.
func (c *Ctx[D]) Eventually(t TestingT, name string, window time.Duration, fn func() error) {
	t.Helper()
	deadline := time.Now().Add(window)
	interval := window / 5
	for {
		err := fn()
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s %s: condition not met within %s: %v", c.prefix, name, window, err)
			return
		}
		time.Sleep(interval)
	}
}
