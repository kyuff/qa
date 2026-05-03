package qa

import "testing"

type Ctx[D any] struct {
	T      *testing.T
	Data   D
	prefix string
}

func (c *Ctx[D]) Run(name string, fn func(*testing.T)) bool {
	return c.T.Run(c.prefix+" "+name, fn)
}
