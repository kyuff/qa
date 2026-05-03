package qa

import "testing"

type Ctx[D any] struct {
	T    *testing.T
	Data D
}
