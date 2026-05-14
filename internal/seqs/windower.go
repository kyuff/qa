package seqs

import (
	"iter"
	"time"
)

func NewWindower(dur time.Duration, maxRetries uint8) iter.Seq[uint8] {
	return func(yield func(uint8) bool) {
		if maxRetries == 0 {
			return
		}
		var (
			deadline = time.Now().Add(dur)
			interval = dur / time.Duration(maxRetries)
		)
		for i := range maxRetries {
			if !yield(i) {
				return
			}
			time.Sleep(interval)
			if time.Now().After(deadline) {
				return
			}
		}
	}
}
