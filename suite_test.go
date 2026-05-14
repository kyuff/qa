package qa_test

import (
	"errors"
	"testing"
	"time"

	"github.com/kyuff/qa"
)

type testData struct {
	CreatedID string
}

type testGiven struct{ *qa.Ctx[*testData] }

func (g *testGiven) SomethingExists() *testGiven {
	g.Run("something exists", func(t *testing.T) {
		g.Data.CreatedID = "abc-123"
	})
	return g
}

type testWhen struct{ *qa.Ctx[*testData] }

func (w *testWhen) ActionHappens() *testWhen {
	w.Run("action happens", func(t *testing.T) {})
	return w
}

type testThen struct{ *qa.Ctx[*testData] }

func (th *testThen) ResultIsAccepted() *testThen {
	th.Run("result is accepted", func(t *testing.T) {
		if th.Data.CreatedID == "" {
			t.Error("expected CreatedID to be set")
		}
	})
	return th
}

var suite = qa.NewSuite(
	func(ctx *qa.Ctx[*testData]) *testGiven { return &testGiven{ctx} },
	func(ctx *qa.Ctx[*testData]) *testWhen { return &testWhen{ctx} },
	func(ctx *qa.Ctx[*testData]) *testThen { return &testThen{ctx} },
	func(t *testing.T) *testData { return &testData{} },
	qa.WithTimeWindow(100*time.Millisecond, 20),
)

func TestSuite(t *testing.T) {
	t.Run("shared ctx allows given to set data that then can read", func(t *testing.T) {
		given, when, then := suite(t)

		given.SomethingExists()
		when.ActionHappens()
		then.ResultIsAccepted()
	})

	t.Run("Eventually", func(t *testing.T) {
		t.Run("should fail when fn never returns nil within window", func(t *testing.T) {
			// arrange
			var (
				failed    = false
				sut, _, _ = suite(t,
					qa.WithTimeWindow(100*time.Millisecond, 20),
					qa.WithFailFunc(func(t *testing.T, format string, args ...any) {
						failed = true
					}),
				)
			)

			// act
			sut.Eventually("condition", func(t *testing.T) error {
				return errors.New("always failing")
			})

			// assert
			if !failed {
				t.Error("expected test to fail when fn never returns nil")
			}
		})

		t.Run("should not fail when fn returns nil on first attempt", func(t *testing.T) {
			// arrange
			var (
				failed    = false
				_, sut, _ = suite(t,
					qa.WithTimeWindow(100*time.Millisecond, 20),
					qa.WithFailFunc(func(t *testing.T, format string, args ...any) {
						failed = true
					}),
				)
			)

			// act
			sut.Eventually("condition", func(t *testing.T) error {
				return nil
			})

			// assert
			if failed {
				t.Error("expected not to fail when fn returns nil")
			}
		})

		t.Run("should not fail when fn eventually returns nil within window", func(t *testing.T) {
			// arrange
			var (
				failed    = false
				calls     = 0
				_, _, sut = suite(t,
					qa.WithTimeWindow(100*time.Millisecond, 20),
					qa.WithFailFunc(func(t *testing.T, format string, args ...any) {
						failed = true
					}),
				)
			)

			// act
			sut.Eventually("condition", func(t *testing.T) error {
				if calls < 3 {
					calls++
					return errors.New("not ready yet")
				}
				return nil
			})

			// assert
			if failed {
				t.Error("expected not to fail when fn returns nil")
			}
		})
	})
}
