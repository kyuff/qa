package qa

import (
	"context"
	"os"
	"os/exec"
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
	a.proc.Stdout = os.Stdout
	a.proc.Stderr = os.Stderr
	if err := a.proc.Start(); err != nil {
		return err
	}
	go func() {
		defer close(a.done)
		a.proc.Wait() //nolint:errcheck
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
