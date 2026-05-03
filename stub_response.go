package qa

// StubResponse is a fluent builder for configuring a stub response.
// Rules are sent to the stub server over HTTP so they work identically
// in local, stubs-only, and ci modes.
type StubResponse struct {
	stub    *HTTPStub
	method  string
	path    string
	matcher Matcher
}

// WithBody narrows this response to requests whose body satisfies the matcher.
// Use this to isolate parallel tests by matching on a unique value per test
// (e.g. an order ID), avoiding resets between tests.
func (r *StubResponse) WithBody(m Matcher) *StubResponse {
	r.matcher = m
	return r
}

// Return registers the configured response with the stub server.
func (r *StubResponse) Return(status int, body string) {
	r.stub.postRule(r.method, r.path, r.matcher, status, body)
}
