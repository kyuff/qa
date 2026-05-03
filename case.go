package qa

import "testing"

type CaseFunc[G, W, T, D any] func(t *testing.T) (G, W, T)

func NewCase[G, W, T, D any](
	given func(*Ctx[D]) G,
	when func(*Ctx[D]) W,
	then func(*Ctx[D]) T,
	data func() D,
) CaseFunc[G, W, T, D] {
	return func(t *testing.T) (G, W, T) {
		ctx := &Ctx[D]{
			T:    t,
			Data: data(),
		}
		return given(ctx), when(ctx), then(ctx)
	}
}
