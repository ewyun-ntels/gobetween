//go:build linux

// Test-only adapter for Docker's non-root --cap-add behavior. NOT shipped as
// an entrypoint and NOT required by gobetween. It drops UID and all effective
// capabilities except BPF, then execs gobetween as PID 1 with no-new-privileges.
package main

import (
	"fmt"
	"os"
	"runtime"
	"syscall"

	"golang.org/x/sys/unix"
)

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "test ambient launcher:", err)
		os.Exit(1)
	}
}

func main() {
	runtime.LockOSThread()
	must(unix.Prctl(unix.PR_SET_KEEPCAPS, 1, 0, 0, 0))
	must(syscall.Setgroups(nil))
	must(syscall.Setresgid(0, 0, 0))
	must(syscall.Setresuid(1001290000, 1001290000, 1001290000))
	header := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
	data := [2]unix.CapUserData{}
	bit := uint32(1) << (unix.CAP_BPF % 32)
	data[unix.CAP_BPF/32] = unix.CapUserData{Effective: bit, Permitted: bit, Inheritable: bit}
	must(unix.Capset(&header, &data[0]))
	must(unix.Prctl(unix.PR_CAP_AMBIENT, unix.PR_CAP_AMBIENT_RAISE, unix.CAP_BPF, 0, 0))
	must(syscall.Exec("/gobetween", append([]string{"/gobetween"}, os.Args[1:]...), os.Environ()))
}
