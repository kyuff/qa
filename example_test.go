package qa_test

// Design sketch — application and payment service are fictional.
// Shows how content-based stub matching isolates parallel tests without resets.

import (
	"fmt"
	"os"
	"testing"

	"github.com/kyuff/qa"
)

var paymentStub = qa.NewHTTPStub("payments")

func TestMain(m *testing.M) {
	rt := qa.NewRuntime(
		qa.WithStub(paymentStub),
		qa.WithApp("./cmd/myapp", "--payment-url="+paymentStub.URL),
	)
	os.Exit(rt.Run(m))
}

// --- Test data ---

type orderData struct {
	OrderID     string
	PaymentStub *qa.HTTPStub
}

var orderSuite = qa.NewSuite(
	func(ctx *qa.Ctx[*orderData]) *orderGiven { return &orderGiven{ctx} },
	func(ctx *qa.Ctx[*orderData]) *orderWhen { return &orderWhen{ctx} },
	func(ctx *qa.Ctx[*orderData]) *orderThen { return &orderThen{ctx} },
	func(t *testing.T) *orderData {
		return &orderData{
			OrderID:     uniqueID(t), // unique per test — the isolation key
			PaymentStub: paymentStub,
		}
	},
)

// uniqueID produces a test-scoped identifier stable within one test run.
func uniqueID(t *testing.T) string {
	return fmt.Sprintf("test-%s", t.Name())
}

// --- Given ---

type orderGiven struct{ ctx *qa.Ctx[*orderData] }

func (g *orderGiven) PaymentServiceAcceptsCharge() *orderGiven {
	g.ctx.Run("payment service accepts charge", func(t *testing.T) {
		g.ctx.Data.PaymentStub.
			On("POST", "/charge").
			WithBody(qa.Contains(g.ctx.Data.OrderID)).
			Return(200, `{"ok":true}`)
	})
	return g
}

// --- When ---

type orderWhen struct{ ctx *qa.Ctx[*orderData] }

func (w *orderWhen) OrderIsPlaced() *orderWhen {
	w.ctx.Run("order is placed", func(t *testing.T) {
		// TODO: POST to application API with OrderID in the request
	})
	return w
}

// --- Then ---

type orderThen struct{ ctx *qa.Ctx[*orderData] }

func (th *orderThen) PaymentWasCharged() *orderThen {
	th.ctx.Run("payment was charged", func(t *testing.T) {
		calls := th.ctx.Data.PaymentStub.
			Calls("POST", "/charge").
			WithBody(qa.Contains(th.ctx.Data.OrderID))
		if len(calls) == 0 {
			t.Errorf("expected payment service to be called with order %s", th.ctx.Data.OrderID)
		}
	})
	return th
}

// --- Tests ---

func TestOrder(t *testing.T) {
	t.Parallel()
	t.Run("placed order charges payment", func(t *testing.T) {
		t.Parallel()
		given, when, then := orderSuite(t)

		given.PaymentServiceAcceptsCharge()
		when.OrderIsPlaced()
		then.PaymentWasCharged()
	})
}
