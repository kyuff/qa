package qa_test

import (
	"testing"

	"github.com/kyuff/qa"
)

// OrderData holds state shared between Given, When, and Then stages.
type OrderData struct {
	OrderID string
}

// OrderGiven sets up preconditions by calling the application under test.
type OrderGiven struct {
	*qa.Ctx[*OrderData]
}

func (g *OrderGiven) FoodProductsInStock() *OrderGiven {
	g.Run("food products in stock", func(t *testing.T) {
		// seed food products via HTTP
	})
	return g
}

// OrderWhen triggers actions on the application under test.
type OrderWhen struct {
	*qa.Ctx[*OrderData]
}

func (w *OrderWhen) CustomerSubmitsBasket() *OrderWhen {
	w.Run("customer submits basket", func(t *testing.T) {
		// POST /basket/submit and store the response
		w.Data.OrderID = "order-001"
	})
	return w
}

// OrderThen asserts on the outcomes received from the application.
type OrderThen struct {
	*qa.Ctx[*OrderData]
}

func (th *OrderThen) OrderIsAccepted() *OrderThen {
	th.Run("order is accepted", func(t *testing.T) {
		if th.Data.OrderID == "" {
			t.Error("expected an order ID in the response")
		}
	})
	return th
}

// orderSuite is defined once per test suite and called with t in each test case.
var orderSuite = qa.NewSuite(
	func(ctx *qa.Ctx[*OrderData]) *OrderGiven { return &OrderGiven{ctx} },
	func(ctx *qa.Ctx[*OrderData]) *OrderWhen { return &OrderWhen{ctx} },
	func(ctx *qa.Ctx[*OrderData]) *OrderThen { return &OrderThen{ctx} },
	func() *OrderData { return &OrderData{} },
)

func ExampleNewSuite() {
	// orderSuite is declared once at package level (see above).
	// Each test case calls it with its own *testing.T:
	//
	//   t.Run("should create an order successfully", func(t *testing.T) {
	//       given, when, then := orderSuite(t)
	//
	//       given.FoodProductsInStock()
	//       when.CustomerSubmitsBasket()
	//       then.OrderIsAccepted()
	//   })
}
