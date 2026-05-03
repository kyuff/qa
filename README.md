# Quality Assurance in Go

[![Build Status](https://github.com/kyuff/qa/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/kyuff/es/actions/workflows/qa.yml)
[![Report Card](https://goreportcard.com/badge/github.com/kyuff/qa)](https://goreportcard.com/report/github.com/kyuff/qa/)
[![Go Reference](https://pkg.go.dev/badge/github.com/kyuff/qa.svg)](https://pkg.go.dev/github.com/kyuff/qa)
[![codecov](https://codecov.io/gh/kyuff/qa/graph/badge.svg?token=EY0LT9XASR)](https://codecov.io/gh/kyuff/qa)

A library for writing blackbox end-to-end tests against a running application using a BDD-style three-layer structure.

## Layers

Tests are organised into three layers:

1. **Test cases** — BDD-style `given/when/then` tests written against the wiring layer.
2. **Wiring layer** — Exposes fluent methods that translate intent into network actions.
3. **Network layer** — Makes HTTP calls, asserts on mocked external services, etc.

## Usage

Define a shared data type and a `Given`, `When`, `Then` struct for each test suite. Each struct embeds `*qa.Ctx[D]` to access `Run` and the shared `Data`.

```go
type Data struct {
    OrderID string
}

type Given struct{ *qa.Ctx[*Data] }

func (g *Given) FoodProductsInStock() *Given {
    g.Run("food products in stock", func(t *testing.T) {
        // seed via HTTP
    })
    return g
}

type When struct{ *qa.Ctx[*Data] }

func (w *When) CustomerSubmitsBasket() *When {
    w.Run("customer submits basket", func(t *testing.T) {
        // POST /basket/submit; store response in w.Data
    })
    return w
}

type Then struct{ *qa.Ctx[*Data] }

func (th *Then) OrderIsAccepted() *Then {
    th.Run("order is accepted", func(t *testing.T) {
        if th.Data.OrderID == "" {
            t.Error("expected an order ID in the response")
        }
    })
    return th
}
```

Wire the suite once at package level:

```go
var suite = qa.NewSuite(
    func(ctx *qa.Ctx[*Data]) *Given { return &Given{ctx} },
    func(ctx *qa.Ctx[*Data]) *When  { return &When{ctx} },
    func(ctx *qa.Ctx[*Data]) *Then  { return &Then{ctx} },
    func() *Data { return &Data{} },
)
```

Call it with `t` in each test case:

```go
func TestOrder(t *testing.T) {
    t.Run("should create an order successfully", func(t *testing.T) {
        given, when, then := suite(t)

        given.FoodProductsInStock()
        when.CustomerSubmitsBasket()
        then.OrderIsAccepted()
    })
}
```

Each stage's `Run` call produces a named sub-test prefixed with the stage name:

```
--- PASS: TestOrder/should_create_an_order_successfully
    --- PASS: .../Given_food_products_in_stock
    --- PASS: .../When_customer_submits_basket
    --- PASS: .../Then_order_is_accepted
```

## Data sharing

`Data` is initialised once per test case via the `data` factory passed to `NewSuite`. Using a pointer type (`*Data`) lets all three stages read and write shared state — for example, `When` can store a created order ID that `Then` later asserts on.
