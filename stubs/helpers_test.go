package stubs_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/kyuff/qa/stubs"
)

func startedHTTP(t *testing.T) *stubs.HTTP {
	t.Helper()
	s := stubs.NewHTTP(t.Name())
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("startedHTTP: %v", err)
	}
	t.Cleanup(func() { s.Stop(context.Background()) })
	return s
}

func post(t *testing.T, url, body string) int {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("post %s: %v", url, err)
	}
	resp.Body.Close()
	return resp.StatusCode
}
