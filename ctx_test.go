package qa_test

import (
	"errors"
	"testing"
	"time"

	"github.com/kyuff/qa"
)

func newTestingTMock() *TestingTMock {
	return &TestingTMock{
		HelperFunc: func() {},
		FatalfFunc: func(format string, args ...any) {},
	}
}

func TestCtx_Eventually(t *testing.T) {
	t.Run("should call Fatalf when fn never returns nil within window", func(t *testing.T) {
		// arrange
		var (
			mock = newTestingTMock()
			sut  = &qa.Ctx[struct{}]{}
		)

		// act
		sut.Eventually(mock, "condition", 50*time.Millisecond, func() error {
			return errors.New("always failing")
		})

		// assert
		if len(mock.FatalfCalls()) == 0 {
			t.Error("expected Fatalf to be called when fn never returns nil")
		}
	})

	t.Run("should not call Fatalf when fn returns nil on first attempt", func(t *testing.T) {
		// arrange
		var (
			mock = newTestingTMock()
			sut  = &qa.Ctx[struct{}]{}
		)

		// act
		sut.Eventually(mock, "condition", time.Second, func() error {
			return nil
		})

		// assert
		if len(mock.FatalfCalls()) != 0 {
			t.Error("expected Fatalf not to be called when fn returns nil")
		}
	})

	t.Run("should not call Fatalf when fn eventually returns nil within window", func(t *testing.T) {
		// arrange
		var (
			mock  = newTestingTMock()
			sut   = &qa.Ctx[struct{}]{}
			calls = 0
		)

		// act
		sut.Eventually(mock, "condition", time.Second, func() error {
			calls++
			if calls < 3 {
				return errors.New("not ready yet")
			}
			return nil
		})

		// assert
		if len(mock.FatalfCalls()) != 0 {
			t.Error("expected Fatalf not to be called when fn eventually returns nil")
		}
		if calls < 3 {
			t.Errorf("expected at least 3 calls, got %d", calls)
		}
	})
}
