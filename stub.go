package qa

// HTTPStub is a controllable HTTP server that records incoming requests
// and returns configured responses. It can run in-process (local dev)
// or connect to an external stub process (CI via docker-compose).
//
// Use NewHTTPStub for local development. Point URL at an existing
// stub server for CI by using ExternalHTTPStub.
type HTTPStub struct {
	// URL is where the application should be configured to call.
	URL  string
	name string
	// TODO: management client or embedded server
}

// NewHTTPStub creates a stub that starts an in-process HTTP server.
func NewHTTPStub(name string) *HTTPStub {
	return &HTTPStub{name: name}
}

// ExternalHTTPStub connects to a stub server already running at the given URL.
// Use this on CI where the stub is started by docker-compose.
func ExternalHTTPStub(name, url string) *HTTPStub {
	return &HTTPStub{name: name, URL: url}
}

// On configures a response for a given method and path.
// Call this in your Given phase before the action under test.
func (s *HTTPStub) On(method, path string) *StubResponse {
	return &StubResponse{stub: s, method: method, path: path}
}

// Calls returns all recorded requests matching the given method and path.
// Call this in your Then phase to assert the application called the dependency.
func (s *HTTPStub) Calls(method, path string) []RecordedCall {
	return nil // TODO: query management API
}

// Reset clears all configured responses and recorded calls.
// Register it via t.Cleanup(stub.Reset) in the data factory to reset between tests.
func (s *HTTPStub) Reset() {
	// TODO: call management API
}

func (s *HTTPStub) start() {
	// TODO: start in-process HTTP server, set s.URL
}

// StubResponse is a fluent builder for configuring a stub response.
type StubResponse struct {
	stub   *HTTPStub
	method string
	path   string
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
