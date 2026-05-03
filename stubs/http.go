package stubs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
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
// configured responses. All management (rule registration, call queries, shutdown)
// uses an HTTP protocol so the same code path runs in every mode.
//
// Use WithAddr to set a fixed host:port. Required for stubs-only and ci modes
// where the application must reach the stub at a known address.
// Omitting WithAddr assigns a random port, which is fine for local-only use.
type HTTP struct {
	// URL is the base address the application should call.
	// Set at construction when WithAddr is used, or after Start returns otherwise.
	URL  string
	name string
	cfg  *config

	mu       sync.RWMutex
	rules    []httpRule
	calls    []RecordedCall
	srv      *http.Server
	done     chan struct{}
	doneOnce sync.Once
}

// NewHTTP creates an HTTP stub. Call Start to bring it up, or register it with
// qa.WithStub which calls Start automatically in local and stubs-only modes.
func NewHTTP(name string, opts ...Option) *HTTP {
	cfg := applyOptions(defaultConfig(), opts...)
	s := &HTTP{
		name: name,
		cfg:  cfg,
		done: make(chan struct{}),
	}
	if cfg.addr != "localhost:0" {
		s.URL = "http://" + cfg.addr
	}
	return s
}

// Start binds the port and starts the HTTP server in the background.
// It returns once the server is ready to accept connections.
func (s *HTTP) Start(_ context.Context) error {
	ln, err := net.Listen("tcp", s.cfg.addr)
	if err != nil {
		return fmt.Errorf("stub %s: listen: %w", s.name, err)
	}
	if s.URL == "" {
		s.URL = "http://" + ln.Addr().String()
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/_qa/rules", s.handleRules)
	mux.HandleFunc("/_qa/calls", s.handleCalls)
	mux.HandleFunc("/_qa/shutdown", s.handleShutdown)
	mux.HandleFunc("/", s.handleApp)
	s.srv = &http.Server{Handler: mux}
	go s.srv.Serve(ln) //nolint:errcheck
	return nil
}

// Stop sends a shutdown signal to the stub over HTTP.
// Safe to call from any mode; also unblocks any concurrent Wait call.
func (s *HTTP) Stop(ctx context.Context) {
	if s.URL == "" {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.URL+"/_qa/shutdown", nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
	s.doneOnce.Do(func() { close(s.done) })
}

// Wait blocks until the stub receives a shutdown signal or ctx is cancelled.
func (s *HTTP) Wait(ctx context.Context) {
	select {
	case <-s.done:
	case <-ctx.Done():
	}
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
	resp, err := http.Get(s.URL + "/_qa/calls?" + q.Encode())
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
	resp, err := http.Post(s.URL+"/_qa/rules", "application/json", bytes.NewReader(data))
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
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
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

func (s *HTTP) handleShutdown(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	s.doneOnce.Do(func() { close(s.done) })
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.srv.Shutdown(ctx) //nolint:errcheck
	}()
}
