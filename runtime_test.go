package qa_test

import (
	"context"
	"testing"

	"github.com/kyuff/qa"
)

type testingMMock struct{ code int }

func (m *testingMMock) Run() int { return m.code }

func TestRun(t *testing.T) {
	t.Run("local mode", func(t *testing.T) {
		t.Run("should start all registered stubs", func(t *testing.T) {
			// arrange
			var (
				stub = &StubMock{
					StartFunc: func(ctx context.Context) error { return nil },
					StopFunc:  func(ctx context.Context) {},
				}
				m = &testingMMock{}
			)

			// act
			qa.Run(m, qa.WithStub(stub))

			// assert
			if len(stub.StartCalls()) != 1 {
				t.Errorf("got %d Start calls, want 1", len(stub.StartCalls()))
			}
		})

		t.Run("should stop all registered stubs after tests", func(t *testing.T) {
			// arrange
			var (
				stub = &StubMock{
					StartFunc: func(ctx context.Context) error { return nil },
					StopFunc:  func(ctx context.Context) {},
				}
				m = &testingMMock{}
			)

			// act
			qa.Run(m, qa.WithStub(stub))

			// assert
			if len(stub.StopCalls()) != 1 {
				t.Errorf("got %d Stop calls, want 1", len(stub.StopCalls()))
			}
		})
	})

	t.Run("ci mode", func(t *testing.T) {
		t.Run("should not start stubs", func(t *testing.T) {
			t.Setenv("QA_MODE", "ci")

			// arrange
			var (
				stub = &StubMock{
					StopFunc: func(ctx context.Context) {},
				}
				m = &testingMMock{}
			)

			// act
			qa.Run(m, qa.WithStub(stub))

			// assert
			if len(stub.StartCalls()) != 0 {
				t.Errorf("got %d Start calls, want 0", len(stub.StartCalls()))
			}
		})

		t.Run("should stop stubs after tests", func(t *testing.T) {
			t.Setenv("QA_MODE", "ci")

			// arrange
			var (
				stub = &StubMock{
					StopFunc: func(ctx context.Context) {},
				}
				m = &testingMMock{}
			)

			// act
			qa.Run(m, qa.WithStub(stub))

			// assert
			if len(stub.StopCalls()) != 1 {
				t.Errorf("got %d Stop calls, want 1", len(stub.StopCalls()))
			}
		})
	})

	t.Run("stubs-only mode", func(t *testing.T) {
		t.Run("should start stubs and wait for shutdown", func(t *testing.T) {
			t.Setenv("QA_MODE", "stubs-only")

			// arrange
			var (
				stub = &StubMock{
					StartFunc: func(ctx context.Context) error { return nil },
					WaitFunc:  func(ctx context.Context) {},
				}
				m = &testingMMock{}
			)

			// act
			code := qa.Run(m, qa.WithStub(stub))

			// assert
			if len(stub.StartCalls()) != 1 {
				t.Errorf("got %d Start calls, want 1", len(stub.StartCalls()))
			}
			if len(stub.WaitCalls()) != 1 {
				t.Errorf("got %d Wait calls, want 1", len(stub.WaitCalls()))
			}
			if code != 0 {
				t.Errorf("got exit code %d, want 0", code)
			}
		})
	})
}
