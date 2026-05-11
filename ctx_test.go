package qa_test

import (
	"errors"
	"testing"
	"time"

	"github.com/kyuff/qa"
)

func TestCtx_Eventually(t *testing.T) {
	t.Run("should fail when fn never returns nil within window", func(t *testing.T) {
		// arrange
		var (
			fatalfCalled = false
			sut          = &qa.Ctx[struct{}]{
				T: t,
				Fatalf: func(format string, args ...any) {
					fatalfCalled = true
				},
			}
		)

		// act
		sut.Eventually("condition", 50*time.Millisecond, func(t *testing.T) error {
			return errors.New("always failing")
		})

		// assert
		if !fatalfCalled {
			t.Error("expected Fatalf to be called when fn never returns nil")
		}
	})

	t.Run("should not fail when fn returns nil on first attempt", func(t *testing.T) {
		// arrange
		var (
			fatalfCalled = false
			sut          = &qa.Ctx[struct{}]{
				T: t,
				Fatalf: func(format string, args ...any) {
					fatalfCalled = true
				},
			}
		)

		// act
		sut.Eventually("condition", time.Second, func(t *testing.T) error {
			return nil
		})

		// assert
		if fatalfCalled {
			t.Error("expected Fatalf not to be called when fn returns nil")
		}
	})

	t.Run("should not fail when fn eventually returns nil within window", func(t *testing.T) {
		// arrange
		var (
			fatalfCalled = false
			calls        = 0
			sut          = &qa.Ctx[struct{}]{
				T: t,
				Fatalf: func(format string, args ...any) {
					fatalfCalled = true
				},
			}
		)

		// act
		sut.Eventually("condition", time.Second, func(t *testing.T) error {
			calls++
			if calls < 3 {
				return errors.New("not ready yet")
			}
			return nil
		})

		// assert
		if fatalfCalled {
			t.Error("expected Fatalf not to be called when fn eventually returns nil")
		}
		if calls < 3 {
			t.Errorf("expected at least 3 calls, got %d", calls)
		}
	})
}
