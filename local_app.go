package qa

import "context"

type appController interface {
	start(ctx context.Context) error
	stop(ctx context.Context)
}

type localApp struct {
	cmd  string
	args []string
}

func (a *localApp) start(_ context.Context) error {
	// TODO: exec.Command, wait for health check
	return nil
}

func (a *localApp) stop(_ context.Context) {
	// TODO: signal process, wait for exit
}
