package qa_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/kyuff/qa"
)

// startedStub starts a local HTTPStub on a random port and registers cleanup.
func startedStub(t *testing.T) *qa.HTTPStub {
	t.Helper()
	s := qa.NewHTTPStub(t.Name())
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("startedStub: %v", err)
	}
	t.Cleanup(func() { s.Stop(context.Background()) })
	return s
}

// post sends a POST with body to url and returns the status code.
func post(t *testing.T, url, body string) int {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("post %s: %v", url, err)
	}
	resp.Body.Close()
	return resp.StatusCode
}
