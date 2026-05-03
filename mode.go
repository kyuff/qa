package qa

import "os"

type runMode string

const (
	runModeLocal     runMode = ""
	runModeStubsOnly runMode = "stubs-only"
	runModeCI        runMode = "ci"
)

func currentRunMode() runMode {
	return runMode(os.Getenv("QA_MODE"))
}
