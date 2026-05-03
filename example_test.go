package qa_test

// This file shows what a test suite looks like when using qa.Runtime.
// The application and payment service are fictional — this is a design sketch.

import (
	"os"
	"testing"

	"github.com/kyuff/qa"
)

// Stubs are package-level so their URLs are reachable from anywhere in the package.
var paymentStub = qa.NewHTTPStub("payments")

func TestMain(m *testing.M) {
	rt := qa.NewRuntime(
		qa.WithHTTPStub(paymentStub), // starts stub; paymentStub.URL is now set
		qa.WithApp("./cmd/myapp", "--payment-url="+paymentStub.URL),
	)
	os.Exit(rt.Run(m))
}

// --- Test data ---

type orderData struct {
	PaymentStub *qa.HTTPStub
}

var orderSuite = qa.NewSuite(
	func(ctx *qa.Ctx[*orderData]) *orderGiven { return &orderGiven{ctx} },
	func(ctx *qa.Ctx[*orderData]) *orderWhen { return &orderWhen{ctx} },
	func(ctx *qa.Ctx[*orderData]) *orderThen { return &orderThen{ctx} },
	func(t *testing.T) *orderData {
		t.Cleanup(paymentStub.Reset) // stubs are reset after each test automatically
		return &orderData{
			PaymentStub: paymentStub,
		}
	},
)

// --- Given ---

type orderGiven struct{ ctx *qa.Ctx[*orderData] }

func (g *orderGiven) PaymentServiceAcceptsCharge() *orderGiven {
	g.ctx.Run("payment service accepts charge", func(t *testing.T) {
		g.ctx.Data.PaymentStub.On("POST", "/charge").Return(200, `{"ok":true}`)
	})
	return g
}

// --- When ---

type orderWhen struct{ ctx *qa.Ctx[*orderData] }

func (w *orderWhen) OrderIsPlaced() *orderWhen {
	w.ctx.Run("order is placed", func(t *testing.T) {
		// TODO: POST to application API
	})
	return w
}

// --- Then ---

type orderThen struct{ ctx *qa.Ctx[*orderData] }

func (th *orderThen) PaymentWasCharged() *orderThen {
	th.ctx.Run("payment was charged", func(t *testing.T) {
		calls := th.ctx.Data.PaymentStub.Calls("POST", "/charge")
		if len(calls) == 0 {
			t.Error("expected payment service to be called")
		}
	})
	return th
}

// --- Tests ---

func TestOrder(t *testing.T) {
	t.Run("placed order charges payment", func(t *testing.T) {
		given, when, then := orderSuite(t)

		given.PaymentServiceAcceptsCharge()
		when.OrderIsPlaced()
		then.PaymentWasCharged()
	})
}
