package qa

import (
	"testing"
)

type Suite[G, W, T, D any] func(t *testing.T, testCaseOpts ...Option) (G, W, T)

// NewSuite wires together the Given/When/Then phases with a per-test data factory.
// The data factory receives *testing.T so it can register cleanups (e.g. stub resets).
func NewSuite[G, W, T, D any](
	given func(*Ctx[D]) G,
	when func(*Ctx[D]) W,
	then func(*Ctx[D]) T,
	data func(*testing.T) D,
	globalOpts ...Option,
) Suite[G, W, T, D] {
	return func(t *testing.T, testCaseOpts ...Option) (G, W, T) {
		var (
			d   = data(t)
			cfg = applyOptions(
				applyOptions(defaultConfig(), globalOpts...),
				testCaseOpts...)
		)
		return given(newCtx(t, "Given", d, cfg)),
			when(newCtx(t, "When", d, cfg)),
			then(newCtx(t, "Then", d, cfg))
	}
}
