package httpstub

import "bytes"

// Matcher evaluates a request body.
type Matcher interface {
	Match(body []byte) bool
}

// Contains returns a Matcher that passes when the body contains s.
func Contains(s string) Matcher {
	return containsMatcher{value: s}
}

type containsMatcher struct{ value string }

func (m containsMatcher) Match(body []byte) bool {
	return len(m.value) > 0 && bytes.Contains(body, []byte(m.value))
}
