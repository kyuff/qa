package qa_test

import (
	"testing"

	"github.com/kyuff/qa"
)

type testData struct {
	CreatedID string
}

type testGiven struct{ ctx *qa.Ctx[*testData] }

func (g *testGiven) SomethingExists() *testGiven {
	g.ctx.Run("something exists", func(t *testing.T) {
		g.ctx.Data.CreatedID = "abc-123"
	})
	return g
}

type testWhen struct{ ctx *qa.Ctx[*testData] }

func (w *testWhen) ActionHappens() *testWhen {
	w.ctx.Run("action happens", func(t *testing.T) {})
	return w
}

type testThen struct{ ctx *qa.Ctx[*testData] }

func (th *testThen) ResultIsAccepted() *testThen {
	th.ctx.Run("result is accepted", func(t *testing.T) {
		if th.ctx.Data.CreatedID == "" {
			t.Error("expected CreatedID to be set")
		}
	})
	return th
}

var suite = qa.NewSuite(
	func(ctx *qa.Ctx[*testData]) *testGiven { return &testGiven{ctx} },
	func(ctx *qa.Ctx[*testData]) *testWhen { return &testWhen{ctx} },
	func(ctx *qa.Ctx[*testData]) *testThen { return &testThen{ctx} },
	func() *testData { return &testData{} },
)

func TestSuite(t *testing.T) {
	t.Run("shared ctx allows given to set data that then can read", func(t *testing.T) {
		given, when, then := suite(t)

		given.SomethingExists()
		when.ActionHappens()
		then.ResultIsAccepted()
	})
}
