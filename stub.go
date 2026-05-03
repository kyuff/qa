package qa

import (
	"bytes"
	"context"
)

// Stub is the lifecycle interface the Runtime uses to manage any stub server.
// Implement this to register a custom protocol stub with NewRuntime.
//
// Start is called in local and stubs-only modes. It is not called in ci mode,
// where stubs are assumed to be already running from a prior stubs-only invocation.
// Stop sends a shutdown signal to the stub. In local and ci modes it is called
// after m.Run completes. In stubs-only mode it is called by the ci-mode binary
// over the network, which causes Wait to unblock.
// Wait blocks until the stub receives its shutdown signal. It is called in
// stubs-only mode so the process keeps running until ci sends Stop.
type Stub interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context)
	Wait(ctx context.Context)
}

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

// RecordedCall holds the details of a single request a stub received.
type RecordedCall struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body"`
}

// RecordedCalls is a filterable slice of recorded requests.
type RecordedCalls []RecordedCall

// WithBody returns only the calls whose body satisfies the matcher.
func (c RecordedCalls) WithBody(m Matcher) RecordedCalls {
	var out RecordedCalls
	for _, call := range c {
		if m.Match(call.Body) {
			out = append(out, call)
		}
	}
	return out
}
