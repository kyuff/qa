package qa_test

// Design sketch — application and payment service are fictional.
// Shows how content-based stub matching isolates parallel tests without resets,
// and how a single TestMain works across local, stubs-only, and ci modes.

import (
	"fmt"
	"os"
	"testing"

	"github.com/kyuff/qa"
	"github.com/kyuff/qa/httpstub"
)

// URL is resolved from the env var or the coded default at package init time,
// so it is ready when WithApp evaluates its arguments.
var paymentStub = httpstub.New(
	httpstub.WithAddrEnv("PAYMENT_STUB_URL"),
)

func TestMain(m *testing.M) {
	os.Exit(qa.Run(m,
		qa.WithStub("payments", paymentStub),
		// WithControlAddr + WithControlAddrEnv required for stubs-only / ci modes.
		// Omitted here so the example runs locally on a random port.
		qa.WithApp("./cmd/myapp", "--payment-url="+paymentStub.URL),
	))
}

// --- Test data ---

type orderData struct {
	OrderID string
}

var orderSuite = qa.NewSuite(
	func(ctx *qa.Ctx[*orderData]) *orderGiven { return &orderGiven{ctx} },
	func(ctx *qa.Ctx[*orderData]) *orderWhen { return &orderWhen{ctx} },
	func(ctx *qa.Ctx[*orderData]) *orderThen { return &orderThen{ctx} },
	func(t *testing.T) *orderData {
		return &orderData{
			OrderID: fmt.Sprintf("order-%s", t.Name()),
		}
	},
)

// --- Given ---

type orderGiven struct{ ctx *qa.Ctx[*orderData] }

func (g *orderGiven) PaymentServiceAcceptsCharge() *orderGiven {
	g.ctx.Run("payment service accepts charge", func(t *testing.T) {
		paymentStub.
			On("POST", "/charge").
			WithBody(httpstub.Contains(g.ctx.Data.OrderID)).
			Return(200, `{"ok":true}`)
	})
	return g
}

// --- When ---

type orderWhen struct{ ctx *qa.Ctx[*orderData] }

func (w *orderWhen) OrderIsPlaced() *orderWhen {
	w.ctx.Run("order is placed", func(t *testing.T) {
		// TODO: POST to application API with OrderID in the request body
	})
	return w
}

// --- Then ---

type orderThen struct{ ctx *qa.Ctx[*orderData] }

func (th *orderThen) PaymentWasCharged() *orderThen {
	th.ctx.Run("payment was charged", func(t *testing.T) {
		calls := paymentStub.
			Calls("POST", "/charge").
			WithBody(httpstub.Contains(th.ctx.Data.OrderID))
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
		_ = then
	})
}
