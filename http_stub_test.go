package qa_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/kyuff/qa"
)

func TestHTTPStub(t *testing.T) {
	t.Run("NewHTTPStub", func(t *testing.T) {
		t.Run("should set URL immediately when addr has fixed port", func(t *testing.T) {
			// arrange
			var (
				addr = "localhost:19099"
			)

			// act
			sut := qa.NewHTTPStub(t.Name(), qa.WithAddr(addr))

			// assert
			if sut.URL != "http://"+addr {
				t.Errorf("got URL %q, want %q", sut.URL, "http://"+addr)
			}
		})

		t.Run("should leave URL empty when using random port", func(t *testing.T) {
			// act
			sut := qa.NewHTTPStub(t.Name())

			// assert
			if sut.URL != "" {
				t.Errorf("expected empty URL before Start, got %q", sut.URL)
			}
		})
	})

	t.Run("Start", func(t *testing.T) {
		t.Run("should populate URL after starting with random port", func(t *testing.T) {
			// arrange
			var (
				sut = qa.NewHTTPStub(t.Name())
			)
			t.Cleanup(func() { sut.Stop(context.Background()) })

			// act
			err := sut.Start(context.Background())

			// assert
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sut.URL == "" {
				t.Error("expected URL to be set after Start")
			}
		})

		t.Run("should fail when port is already in use", func(t *testing.T) {
			// arrange
			var (
				first = qa.NewHTTPStub("first", qa.WithAddr("localhost:19098"))
			)
			if err := first.Start(context.Background()); err != nil {
				t.Fatalf("arrange: %v", err)
			}
			t.Cleanup(func() { first.Stop(context.Background()) })

			sut := qa.NewHTTPStub("second", qa.WithAddr("localhost:19098"))

			// act
			err := sut.Start(context.Background())

			// assert
			if err == nil {
				t.Error("expected error when port is already in use")
				sut.Stop(context.Background())
			}
		})
	})

	t.Run("On", func(t *testing.T) {
		t.Run("should return 404 when no rule matches", func(t *testing.T) {
			// arrange
			var (
				sut = startedStub(t)
			)

			// act
			got := post(t, sut.URL+"/charge", `{}`)

			// assert
			if got != http.StatusNotFound {
				t.Errorf("got status %d, want 404", got)
			}
		})

		t.Run("should return configured status and body", func(t *testing.T) {
			// arrange
			var (
				sut        = startedStub(t)
				wantStatus = http.StatusOK
				wantBody   = `{"ok":true}`
			)
			sut.On("POST", "/charge").Return(wantStatus, wantBody)

			// act
			resp, err := http.Post(sut.URL+"/charge", "application/json", nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			// assert
			if resp.StatusCode != wantStatus {
				t.Errorf("got status %d, want %d", resp.StatusCode, wantStatus)
			}
		})

		t.Run("should return 404 when method does not match", func(t *testing.T) {
			// arrange
			var (
				sut = startedStub(t)
			)
			sut.On("POST", "/charge").Return(http.StatusOK, `{}`)

			// act
			resp, err := http.Get(sut.URL + "/charge")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			resp.Body.Close()

			// assert
			if resp.StatusCode != http.StatusNotFound {
				t.Errorf("got status %d, want 404", resp.StatusCode)
			}
		})

		t.Run("should match when body satisfies WithBody matcher", func(t *testing.T) {
			// arrange
			var (
				sut     = startedStub(t)
				orderID = t.Name()
			)
			sut.On("POST", "/charge").WithBody(qa.Contains(orderID)).Return(http.StatusOK, `{}`)

			// act
			got := post(t, sut.URL+"/charge", `{"id":"`+orderID+`"}`)

			// assert
			if got != http.StatusOK {
				t.Errorf("got status %d, want 200", got)
			}
		})

		t.Run("should return 404 when body does not satisfy WithBody matcher", func(t *testing.T) {
			// arrange
			var (
				sut = startedStub(t)
			)
			sut.On("POST", "/charge").WithBody(qa.Contains("order-A")).Return(http.StatusOK, `{}`)

			// act
			got := post(t, sut.URL+"/charge", `{"id":"order-B"}`)

			// assert
			if got != http.StatusNotFound {
				t.Errorf("got status %d, want 404", got)
			}
		})

		t.Run("should match first rule when multiple rules match", func(t *testing.T) {
			// arrange
			var (
				sut = startedStub(t)
			)
			sut.On("POST", "/charge").Return(http.StatusOK, `{"first":true}`)
			sut.On("POST", "/charge").Return(http.StatusAccepted, `{"second":true}`)

			// act
			got := post(t, sut.URL+"/charge", `{}`)

			// assert
			if got != http.StatusOK {
				t.Errorf("got status %d, want 200 (first rule)", got)
			}
		})
	})

	t.Run("Calls", func(t *testing.T) {
		t.Run("should record calls", func(t *testing.T) {
			// arrange
			var (
				sut = startedStub(t)
			)
			post(t, sut.URL+"/charge", `{}`)
			post(t, sut.URL+"/charge", `{}`)

			// act
			got := sut.Calls("POST", "/charge")

			// assert
			if len(got) != 2 {
				t.Errorf("got %d calls, want 2", len(got))
			}
		})

		t.Run("should filter by method and path", func(t *testing.T) {
			// arrange
			var (
				sut = startedStub(t)
			)
			post(t, sut.URL+"/charge", `{}`)
			http.Get(sut.URL + "/status") //nolint:errcheck

			// act
			got := sut.Calls("POST", "/charge")

			// assert
			if len(got) != 1 {
				t.Errorf("got %d calls, want 1", len(got))
			}
		})

		t.Run("WithBody should filter recorded calls by content", func(t *testing.T) {
			// arrange
			var (
				sut     = startedStub(t)
				orderID = t.Name()
			)
			post(t, sut.URL+"/charge", `{"id":"`+orderID+`"}`)
			post(t, sut.URL+"/charge", `{"id":"other"}`)

			// act
			got := sut.Calls("POST", "/charge").WithBody(qa.Contains(orderID))

			// assert
			if len(got) != 1 {
				t.Errorf("got %d filtered calls, want 1", len(got))
			}
		})
	})

	t.Run("Stop and Wait", func(t *testing.T) {
		t.Run("Wait should unblock after Stop is called", func(t *testing.T) {
			// arrange
			var (
				sut  = qa.NewHTTPStub(t.Name())
				done = make(chan struct{})
			)
			if err := sut.Start(context.Background()); err != nil {
				t.Fatalf("Start: %v", err)
			}

			go func() {
				sut.Wait(context.Background())
				close(done)
			}()

			// act
			sut.Stop(context.Background())

			// assert
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Error("Wait did not unblock after Stop")
			}
		})

		t.Run("Wait should unblock when ctx is cancelled", func(t *testing.T) {
			// arrange
			var (
				sut         = startedStub(t)
				ctx, cancel = context.WithCancel(context.Background())
				done        = make(chan struct{})
			)

			go func() {
				sut.Wait(ctx)
				close(done)
			}()

			// act
			cancel()

			// assert
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Error("Wait did not unblock after ctx cancel")
			}
		})
	})

	t.Run("shutdown protocol", func(t *testing.T) {
		t.Run("ci-mode binary can shut down stubs-only stub over HTTP", func(t *testing.T) {
			// arrange: "stubs-only" stub is just a running server
			var (
				server = qa.NewHTTPStub(t.Name())
				done   = make(chan struct{})
			)
			if err := server.Start(context.Background()); err != nil {
				t.Fatalf("Start: %v", err)
			}

			go func() {
				server.Wait(context.Background())
				close(done)
			}()

			// act: "ci" binary creates a client stub pointing at the same URL
			client := qa.NewHTTPStub(t.Name(), qa.WithAddr(server.URL[7:])) // strip "http://"
			client.Stop(context.Background())

			// assert: the server-side Wait unblocked
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Error("server Wait did not unblock after client Stop")
			}
		})
	})
}
