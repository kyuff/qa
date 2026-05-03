package qa

// Stub is the lifecycle interface the Runtime needs from any stub server.
// Implement this to integrate a custom protocol stub with the Runtime.
type Stub interface {
	Start() error
	Stop()
}

// HTTPStub is a controllable HTTP server that records incoming requests
// and returns configured responses. It is shared across all tests — isolation
// is achieved by matching on request content (e.g. unique IDs in the body),
// not by resetting state between tests.
type HTTPStub struct {
	// URL is where the application should be configured to call.
	// Populated after Start() is called.
	URL  string
	name string
	// TODO: management client or embedded server
}

func NewHTTPStub(name string) *HTTPStub {
	return &HTTPStub{name: name}
}

func ExternalHTTPStub(name, url string) *HTTPStub {
	return &HTTPStub{name: name, URL: url}
}

func (s *HTTPStub) Start() error {
	// TODO: start in-process HTTP server, set s.URL
	return nil
}

func (s *HTTPStub) Stop() {
	// TODO: stop server
}

// On begins configuring a response for a given method and path.
func (s *HTTPStub) On(method, path string) *StubResponse {
	return &StubResponse{stub: s, method: method, path: path}
}

// Calls returns all recorded requests matching the given method and path.
func (s *HTTPStub) Calls(method, path string) RecordedCalls {
	return nil // TODO: query management API
}

// StubResponse is a fluent builder for configuring a stub response.
type StubResponse struct {
	stub    *HTTPStub
	method  string
	path    string
	matcher Matcher
}

// WithBody narrows this response to requests whose body satisfies the matcher.
func (r *StubResponse) WithBody(m Matcher) *StubResponse {
	r.matcher = m
	return r
}

func (r *StubResponse) Return(status int, body string) {
	// TODO: register via management API
}

// RecordedCall holds the details of a single request the stub received.
type RecordedCall struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    []byte
}

// RecordedCalls is a filterable slice of recorded requests.
type RecordedCalls []RecordedCall

// WithBody returns only the calls whose body satisfies the matcher.
func (c RecordedCalls) WithBody(m Matcher) RecordedCalls {
	var out RecordedCalls
	for _, call := range c {
		if m(call.Body) {
			out = append(out, call)
		}
	}
	return out
}

// Matcher is a predicate over a request body.
type Matcher func(body []byte) bool

// Contains returns a Matcher that passes when the body contains s.
func Contains(s string) Matcher {
	return func(body []byte) bool {
		return len(s) > 0 && contains(body, []byte(s))
	}
}

func contains(haystack, needle []byte) bool {
	if len(needle) > len(haystack) {
		return false
	}
	for i := range haystack[:len(haystack)-len(needle)+1] {
		if string(haystack[i:i+len(needle)]) == string(needle) {
			return true
		}
	}
	return false
}
