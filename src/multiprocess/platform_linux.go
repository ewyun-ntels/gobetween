//go:build linux

package multiprocess

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

func platformSupported() bool {
	return true
}

func socketPair() (*os.File, *os.File, error) {
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, nil, err
	}
	return os.NewFile(uintptr(fds[0]), "gobetween-parent-ipc"), os.NewFile(uintptr(fds[1]), "gobetween-worker-ipc"), nil
}

func configureWorkerCommand(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
}

func allowedCPUs() ([]int, error) {
	var set unix.CPUSet
	if err := unix.SchedGetaffinity(0, &set); err != nil {
		return nil, err
	}
	cpus := make([]int, 0, set.Count())
	for cpu := 0; cpu < 1024; cpu++ {
		if set.IsSet(cpu) {
			cpus = append(cpus, cpu)
		}
	}
	return cpus, nil
}

// ExecWorkerWithAffinity pins the bootstrap before exec. The real Go runtime
// therefore starts with the intended affinity mask and GOMAXPROCS value.
func ExecWorkerWithAffinity() error {
	cpus, err := parseCPUList(os.Getenv(envWorkerCPUs))
	if err != nil {
		return err
	}

	runtime.LockOSThread()
	var set unix.CPUSet
	set.Zero()
	for _, cpu := range cpus {
		set.Set(cpu)
	}
	if err := unix.SchedSetaffinity(0, &set); err != nil {
		return fmt.Errorf("set worker CPU affinity: %w", err)
	}

	executable, err := os.Executable()
	if err != nil {
		return err
	}
	env := replaceEnv(os.Environ(), envRole, roleWorker)
	env = replaceEnv(env, "GOMAXPROCS", strconv.Itoa(len(cpus)))
	return unix.Exec(executable, os.Args, env)
}
