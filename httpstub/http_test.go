package httpstub_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/kyuff/qa/httpstub"
)

func TestHTTP(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		t.Run("should set URL immediately when addr has fixed port", func(t *testing.T) {
			// arrange
			var (
				addr = "localhost:19099"
			)

			// act
			sut := httpstub.New(httpstub.WithAddr(addr))

			// assert
			if sut.URL != "http://"+addr {
				t.Errorf("got URL %q, want %q", sut.URL, "http://"+addr)
			}
		})

		t.Run("should leave URL empty when using random port", func(t *testing.T) {
			// act
			sut := httpstub.New()

			// assert
			if sut.URL != "" {
				t.Errorf("expected empty URL before Start, got %q", sut.URL)
			}
		})

		t.Run("should prefer env var over WithAddr when env var is set", func(t *testing.T) {
			// arrange
			t.Setenv("TEST_STUB_ADDR", "payments:19001")

			// act
			sut := httpstub.New(
				httpstub.WithAddr("localhost:19001"),
				httpstub.WithAddrEnv("TEST_STUB_ADDR"),
			)

			// assert
			if sut.URL != "http://payments:19001" {
				t.Errorf("got URL %q, want %q", sut.URL, "http://payments:19001")
			}
		})

		t.Run("should use WithAddr when env var is not set", func(t *testing.T) {
			// arrange — ensure the env var is absent
			t.Setenv("TEST_STUB_ADDR_UNSET", "")

			// act
			sut := httpstub.New(
				httpstub.WithAddr("localhost:19001"),
				httpstub.WithAddrEnv("TEST_STUB_ADDR_UNSET"),
			)

			// assert
			if sut.URL != "http://localhost:19001" {
				t.Errorf("got URL %q, want %q", sut.URL, "http://localhost:19001")
			}
		})
	})

	t.Run("Start", func(t *testing.T) {
		t.Run("should populate URL after starting with random port", func(t *testing.T) {
			// arrange
			var (
				sut  = httpstub.New()
				errc = make(chan error, 1)
			)
			go func() { errc <- sut.Start(context.Background()) }()
			t.Cleanup(func() { sut.Stop(context.Background()) })

			// act — wait for ready
			select {
			case err := <-errc:
				t.Fatalf("Start returned early: %v", err)
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for stub to be ready")
			default:
			}
			waitReady(t, sut)

			// assert
			if sut.URL == "" {
				t.Error("expected URL to be set after Start")
			}
		})

		t.Run("should fail when port is already in use", func(t *testing.T) {
			// arrange
			var (
				first = httpstub.New(httpstub.WithAddr("localhost:19098"))
				errc  = make(chan error, 1)
			)
			go func() { errc <- first.Start(context.Background()) }()
			waitReady(t, first)
			t.Cleanup(func() { first.Stop(context.Background()) })

			sut := httpstub.New(httpstub.WithAddr("localhost:19098"))

			// act
			startErrc := make(chan error, 1)
			go func() { startErrc <- sut.Start(context.Background()) }()

			// assert
			select {
			case err := <-startErrc:
				if err == nil {
					t.Error("expected error when port is already in use")
				}
			case <-time.After(time.Second):
				t.Error("expected Start to fail quickly for port already in use")
				sut.Stop(context.Background())
			}
		})
	})

	t.Run("Probe", func(t *testing.T) {
		t.Run("should return error before Start is called", func(t *testing.T) {
			// act
			sut := httpstub.New()

			// assert
			if sut.Probe() == nil {
				t.Error("expected Probe to return error before Start")
			}
		})

		t.Run("should return nil once server is ready", func(t *testing.T) {
			// arrange
			sut := startedHTTP(t)

			// act + assert
			if err := sut.Probe(); err != nil {
				t.Errorf("expected Probe to return nil after Start, got %v", err)
			}
		})
	})

	t.Run("On", func(t *testing.T) {
		t.Run("should return 404 when no rule matches", func(t *testing.T) {
			// arrange
			var (
				sut = startedHTTP(t)
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
				sut        = startedHTTP(t)
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
				sut = startedHTTP(t)
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
				sut     = startedHTTP(t)
				orderID = t.Name()
			)
			sut.On("POST", "/charge").WithBody(httpstub.Contains(orderID)).Return(http.StatusOK, `{}`)

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
				sut = startedHTTP(t)
			)
			sut.On("POST", "/charge").WithBody(httpstub.Contains("order-A")).Return(http.StatusOK, `{}`)

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
				sut = startedHTTP(t)
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
				sut = startedHTTP(t)
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
				sut = startedHTTP(t)
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
				sut     = startedHTTP(t)
				orderID = t.Name()
			)
			post(t, sut.URL+"/charge", `{"id":"`+orderID+`"}`)
			post(t, sut.URL+"/charge", `{"id":"other"}`)

			// act
			got := sut.Calls("POST", "/charge").WithBody(httpstub.Contains(orderID))

			// assert
			if len(got) != 1 {
				t.Errorf("got %d filtered calls, want 1", len(got))
			}
		})
	})
}

func waitReady(t *testing.T, s *httpstub.HTTP) {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for stub to be ready")
		default:
		}
		if s.Probe() == nil {
			return
		}
		time.Sleep(time.Millisecond)
	}
}
