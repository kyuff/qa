package httpstub

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
)

type httpRule struct {
	method  string
	path    string
	matcher Matcher
	status  int
	body    string
}

type ruleRequest struct {
	Method       string `json:"method"`
	Path         string `json:"path"`
	BodyContains string `json:"body_contains,omitempty"`
	Status       int    `json:"status"`
	ResponseBody string `json:"response_body"`
}

// HTTP is a controllable HTTP server that records incoming requests and returns
// configured responses. Rules and call queries are routed through the qa control
// server, so the same code path works in local, stubs-only, and ci modes.
//
// Construct with New and register with qa.WithStub. The runtime sets the
// management URL before tests run; On and Calls use it automatically.
type HTTP struct {
	// URL is the address the application under test should be configured to call.
	// Set at construction when WithAddr (or WithAddrEnv) resolves to a fixed port,
	// or after Start binds when using a random port.
	URL string

	cfg           *config
	managementURL string

	mu    sync.RWMutex
	rules []httpRule
	calls []RecordedCall
	srv   *http.Server

	ready     chan struct{}
	readyOnce sync.Once
}

// New creates an HTTP stub. Register it with qa.WithStub; the runtime calls
// Start automatically in local and stubs-only modes.
func New(opts ...Option) *HTTP {
	cfg := applyOptions(defaultConfig(), opts...)
	s := &HTTP{
		cfg:   cfg,
		ready: make(chan struct{}),
	}
	addr := cfg.resolveAddr()
	_, port, err := net.SplitHostPort(addr)
	if err == nil && port != "0" {
		s.URL = "http://" + addr
	}
	return s
}

// SetManagementURL is called by the qa runtime to point On and Calls at the
// correct path on the control server. Not part of the public API.
func (s *HTTP) SetManagementURL(u string) {
	s.managementURL = u
}

// Start binds the port, signals readiness, then blocks until Stop is called.
// The port is extracted from the resolved address; the server listens on all
// interfaces so Docker networking can reach it when the hostname differs from
// localhost.
func (s *HTTP) Start(_ context.Context) error {
	addr := s.cfg.resolveAddr()
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("httpstub: invalid addr %q: %w", addr, err)
	}

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("httpstub: listen :%s: %w", port, err)
	}

	if s.URL == "" {
		s.URL = "http://" + ln.Addr().String()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleApp)
	s.srv = &http.Server{Handler: mux}

	s.readyOnce.Do(func() { close(s.ready) })

	if err := s.srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Stop shuts down the HTTP server, causing Start to return.
func (s *HTTP) Stop(ctx context.Context) {
	if s.srv == nil {
		return
	}
	s.srv.Shutdown(ctx) //nolint:errcheck
}

// Probe returns nil once the server has bound its port and is ready to serve.
func (s *HTTP) Probe() error {
	select {
	case <-s.ready:
		return nil
	default:
		return errors.New("not ready")
	}
}

// Handler returns the management http.Handler mounted by the qa control server.
// It exposes POST /rules and GET /calls for the test process to register
// stub behaviour and query recorded calls.
func (s *HTTP) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /rules", s.handleRules)
	mux.HandleFunc("GET /calls", s.handleCalls)
	return mux
}

// On begins configuring a response for the given method and path.
func (s *HTTP) On(method, path string) *Response {
	return &Response{stub: s, method: method, path: path}
}

// Calls returns all recorded requests matching the given method and path.
func (s *HTTP) Calls(method, path string) RecordedCalls {
	q := url.Values{}
	q.Set("method", method)
	q.Set("path", path)
	resp, err := http.Get(s.managementURL + "/calls?" + q.Encode())
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	out := make(RecordedCalls, 0)
	json.NewDecoder(resp.Body).Decode(&out) //nolint:errcheck
	return out
}

func (s *HTTP) postRule(method, path string, matcher Matcher, status int, body string) {
	req := ruleRequest{
		Method:       method,
		Path:         path,
		Status:       status,
		ResponseBody: body,
	}
	if c, ok := matcher.(containsMatcher); ok {
		req.BodyContains = c.value
	}
	data, _ := json.Marshal(req)
	resp, err := http.Post(s.managementURL+"/rules", "application/json", bytes.NewReader(data))
	if err == nil {
		resp.Body.Close()
	}
}

func (s *HTTP) handleApp(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	headers := make(map[string]string, len(r.Header))
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	s.mu.Lock()
	s.calls = append(s.calls, RecordedCall{
		Method:  r.Method,
		Path:    r.URL.Path,
		Headers: headers,
		Body:    body,
	})
	s.mu.Unlock()

	s.mu.RLock()
	var matched *httpRule
	for i := range s.rules {
		rule := &s.rules[i]
		if rule.method == r.Method && rule.path == r.URL.Path {
			if rule.matcher == nil || rule.matcher.Match(body) {
				matched = rule
				break
			}
		}
	}
	s.mu.RUnlock()

	if matched == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(matched.status)
	io.WriteString(w, matched.body) //nolint:errcheck
}

func (s *HTTP) handleRules(w http.ResponseWriter, r *http.Request) {
	var req ruleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var matcher Matcher
	if req.BodyContains != "" {
		matcher = Contains(req.BodyContains)
	}
	s.mu.Lock()
	s.rules = append(s.rules, httpRule{
		method:  req.Method,
		path:    req.Path,
		matcher: matcher,
		status:  req.Status,
		body:    req.ResponseBody,
	})
	s.mu.Unlock()
	w.WriteHeader(http.StatusCreated)
}

func (s *HTTP) handleCalls(w http.ResponseWriter, r *http.Request) {
	method := r.URL.Query().Get("method")
	path := r.URL.Query().Get("path")

	s.mu.RLock()
	out := make(RecordedCalls, 0)
	for _, call := range s.calls {
		if call.Method == method && call.Path == path {
			out = append(out, call)
		}
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out) //nolint:errcheck
}
