package qa_test

import (
	"context"
	"testing"

	"github.com/kyuff/qa"
)

func TestRuntime(t *testing.T) {
	t.Run("NewRuntime", func(t *testing.T) {
		t.Run("should start all registered stubs", func(t *testing.T) {
			// arrange
			var (
				stub = qa.NewHTTPStub(t.Name())
			)
			t.Cleanup(func() { stub.Stop(context.Background()) })

			// act
			_ = qa.NewRuntime(qa.WithStub(stub))

			// assert
			if stub.URL == "" {
				t.Error("expected stub URL to be set after NewRuntime")
			}
		})

		t.Run("should not start stubs in ci mode", func(t *testing.T) {
			t.Setenv("QA_MODE", "ci")

			// arrange
			var (
				stub = qa.NewHTTPStub(t.Name())
			)

			// act
			_ = qa.NewRuntime(qa.WithStub(stub))

			// assert: URL is still empty because Start was never called
			if stub.URL != "" {
				t.Errorf("expected URL to be empty in ci mode, got %q", stub.URL)
			}
		})

		t.Run("should set URL immediately for fixed-addr stub in ci mode", func(t *testing.T) {
			t.Setenv("QA_MODE", "ci")

			// arrange
			var (
				addr = "localhost:19097"
				stub = qa.NewHTTPStub(t.Name(), qa.WithAddr(addr))
			)

			// act
			_ = qa.NewRuntime(qa.WithStub(stub))

			// assert
			if stub.URL != "http://"+addr {
				t.Errorf("got URL %q, want %q", stub.URL, "http://"+addr)
			}
		})
	})
}
