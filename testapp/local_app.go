package testapp

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

type appController interface {
	start(ctx context.Context) error
	stop(ctx context.Context)
}

type localApp struct {
	cmd  string
	args []string
	proc *exec.Cmd
	done chan struct{}
}

func (a *localApp) start(_ context.Context) error {
	a.done = make(chan struct{})
	a.proc = exec.Command(a.cmd, a.args...) //nolint:gosec

	// Run from the module root so that relative paths like ./cmd/server/main.go
	// resolve correctly regardless of which package directory go test runs from.
	if dir := moduleRoot(); dir != "" {
		a.proc.Dir = dir
	}

	// Use pipes instead of os.Stdout/os.Stderr directly. If the subprocess
	// keeps an inherited FD open after the test exits, the Go test runner
	// reports "Test I/O incomplete". Pipes break that reference: the parent
	// closes its write-end after Start, so the read-end reaches EOF as soon
	// as the child exits.
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return err
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		stdoutR.Close()
		stdoutW.Close()
		return err
	}
	a.proc.Stdout = stdoutW
	a.proc.Stderr = stderrW

	if err := a.proc.Start(); err != nil {
		stdoutR.Close()
		stdoutW.Close()
		stderrR.Close()
		stderrW.Close()
		return err
	}

	// Close write-ends in the parent so reads reach EOF when the child exits.
	stdoutW.Close()
	stderrW.Close()

	go func() {
		defer close(a.done)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			io.Copy(os.Stdout, stdoutR) //nolint:errcheck
			stdoutR.Close()
		}()
		go func() {
			defer wg.Done()
			io.Copy(os.Stderr, stderrR) //nolint:errcheck
			stderrR.Close()
		}()
		a.proc.Wait() //nolint:errcheck
		wg.Wait()
	}()
	return nil
}

func (a *localApp) stop(ctx context.Context) {
	if a.proc == nil || a.proc.Process == nil {
		return
	}
	a.proc.Process.Signal(os.Interrupt) //nolint:errcheck
	select {
	case <-a.done:
	case <-ctx.Done():
		a.proc.Process.Kill() //nolint:errcheck
		<-a.done
	}
}

// moduleRoot walks up from the current directory to find the nearest go.mod.
func moduleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
