//go:build !linux

package multiprocess

import (
	"errors"
	"os"
	"os/exec"
)

func platformSupported() bool {
	return false
}

func socketPair() (*os.File, *os.File, error) {
	return nil, nil, errors.New("UDP multiprocess runtime requires Linux")
}

func configureWorkerCommand(command *exec.Cmd) {}

func allowedCPUs() ([]int, error) {
	return nil, errors.New("CPU affinity requires Linux")
}

func ExecWorkerWithAffinity() error {
	return errors.New("UDP multiprocess runtime requires Linux")
}
