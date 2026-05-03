package stubs

// RecordedCall holds the details of a single request the stub received.
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
