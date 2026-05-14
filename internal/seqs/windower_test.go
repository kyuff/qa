package seqs_test

import (
	"testing"
	"time"

	"github.com/kyuff/qa/internal/seqs"
)

func TestNewWindower(t *testing.T) {
	t.Run("yields values from 0 to maxRetries-1", func(t *testing.T) {
		// arrange
		var got []uint8

		// act
		for i := range seqs.NewWindower(time.Second, 4) {
			got = append(got, i)
		}

		// assert
		want := []uint8{0, 1, 2, 3}
		if len(got) != len(want) {
			t.Fatalf("expected %d values, got %d: %v", len(want), len(got), got)
		}
		for i, v := range got {
			if v != want[i] {
				t.Errorf("got[%d] = %d, want %d", i, v, want[i])
			}
		}
	})

	t.Run("stops iteration when yield returns false", func(t *testing.T) {
		// arrange
		var got []uint8

		// act
		for i := range seqs.NewWindower(time.Second, 10) {
			got = append(got, i)
			if i == 2 {
				break
			}
		}

		// assert
		if len(got) != 3 {
			t.Fatalf("expected 3 values, got %d: %v", len(got), got)
		}
	})

	t.Run("stops before maxRetries when deadline is exceeded", func(t *testing.T) {
		// arrange
		var count int

		// act
		for range seqs.NewWindower(10*time.Millisecond, 100) {
			time.Sleep(5 * time.Microsecond)
			count++
		}

		// assert
		if count >= 100 {
			t.Errorf("expected fewer than 100 yields due to deadline, got %d", count)
		}
	})

	t.Run("yields single value when maxRetries is 1", func(t *testing.T) {
		// arrange
		var got []uint8

		// act
		for i := range seqs.NewWindower(time.Second, 1) {
			got = append(got, i)
		}

		// assert
		if len(got) != 1 || got[0] != 0 {
			t.Errorf("expected [0], got %v", got)
		}
	})

	t.Run("yields no value when maxRetries is 0", func(t *testing.T) {
		// arrange
		var got []uint8

		// act
		for i := range seqs.NewWindower(time.Second, 0) {
			got = append(got, i)
		}

		// assert
		if len(got) != 0 {
			t.Errorf("expected [0], got %v", got)
		}
	})
}
