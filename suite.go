package qa

import "testing"

type Suite[G, W, T, D any] func(t *testing.T) (G, W, T)

func NewSuite[G, W, T, D any](
	given func(*Ctx[D]) G,
	when func(*Ctx[D]) W,
	then func(*Ctx[D]) T,
	data func() D,
) Suite[G, W, T, D] {
	return func(t *testing.T) (G, W, T) {
		d := data()
		return given(&Ctx[D]{T: t, Data: d, prefix: "Given"}),
			when(&Ctx[D]{T: t, Data: d, prefix: "When"}),
			then(&Ctx[D]{T: t, Data: d, prefix: "Then"})
	}
}
