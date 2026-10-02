//go:build linux

package multiprocess

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"syscall"
	"testing"
	"unsafe"
)

func TestTopologyCPUList(t *testing.T) {
	for _, test := range []struct {
		value string
		want  []int
	}{
		{"0,96\n", []int{0, 96}},
		{"0-3, 8-9,14,17\n", []int{0, 1, 2, 3, 8, 9, 14, 17}},
		{"48,0", []int{48, 0}},
		{"1023", []int{1023}},
	} {
		got, err := parseTopologyCPUList(test.value)
		if err != nil || !reflect.DeepEqual(got, test.want) {
			t.Fatalf("parse %q=%v err=%v", test.value, got, err)
		}
	}
	for _, value := range []string{"", "\n", "0,", "-1", "+1", "a", "3-1", "1-2-3", "1-", "1024", "0-999999999", "9999999999999999999999", "0,0", "0-2,2"} {
		if _, err := parseTopologyCPUList(value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
}

// Exercises the real sysfs + affinity boundary on the calling test thread only.
// The original mask is restored; no node or container configuration is changed.
func TestLivePhysicalCPUAllocation(t *testing.T) {
	if os.Getenv("GOBETWEEN_TEST_CPU_AFFINITY") != "1" {
		t.Skip("set GOBETWEEN_TEST_CPU_AFFINITY=1 to test real sysfs and thread affinity")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	var original [128]byte
	_, _, errno := syscall.RawSyscall(syscall.SYS_SCHED_GETAFFINITY, 0, uintptr(len(original)), uintptr(unsafe.Pointer(&original[0])))
	if errno != 0 {
		t.Fatal(errno)
	}
	available := []int{}
	for cpu := 0; cpu < len(original)*8; cpu++ {
		if original[cpu/8]&(1<<uint(cpu%8)) != 0 {
			available = append(available, cpu)
		}
	}
	partitions, err := partitionWorkerCPUs(available, 1, "physical", readCoreCPUs)
	if err != nil {
		t.Fatalf("live topology allocation: %v", err)
	}
	if !reflect.DeepEqual(partitions[0], available) {
		t.Fatalf("live allocation lost CPUs: %v versus %v", partitions[0], available)
	}
	group, err := readCoreCPUs(available[0])
	if err != nil {
		t.Fatal(err)
	}
	var mask [128]byte
	for _, cpu := range group {
		mask[cpu/8] |= 1 << uint(cpu%8)
	}
	_, _, errno = syscall.RawSyscall(syscall.SYS_SCHED_SETAFFINITY, 0, uintptr(len(mask)), uintptr(unsafe.Pointer(&mask[0])))
	if errno != 0 {
		t.Fatal(errno)
	}
	defer func() {
		_, _, restoreErr := syscall.RawSyscall(syscall.SYS_SCHED_SETAFFINITY, 0, uintptr(len(original)), uintptr(unsafe.Pointer(&original[0])))
		if restoreErr != 0 {
			t.Errorf("restore affinity: %v", restoreErr)
		}
	}()
	var actual [128]byte
	_, _, errno = syscall.RawSyscall(syscall.SYS_SCHED_GETAFFINITY, 0, uintptr(len(actual)), uintptr(unsafe.Pointer(&actual[0])))
	if errno != 0 || actual != mask {
		t.Fatalf("affinity mismatch: errno=%v", errno)
	}
	t.Logf("allowed logical CPUs=%d; first physical core=%v; live topology and thread affinity verified", len(available), group)
}

func TestReadCoreCPUsAt(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "cpu0", "topology")
	if err := os.MkdirAll(path, 0700); err != nil {
		t.Fatal(err)
	}
	write := func(name, value string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(path, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := readCoreCPUsAt(root, 0); err == nil {
		t.Fatal("missing topology accepted")
	}
	write("thread_siblings_list", "0,48\n")
	if got, err := readCoreCPUsAt(root, 0); err != nil || !reflect.DeepEqual(got, []int{0, 48}) {
		t.Fatalf("fallback=%v err=%v", got, err)
	}
	write("core_cpus_list", "0,96\n")
	if got, err := readCoreCPUsAt(root, 0); err != nil || !reflect.DeepEqual(got, []int{0, 96}) {
		t.Fatalf("preferred=%v err=%v", got, err)
	}
	write("core_cpus_list", "bad")
	if _, err := readCoreCPUsAt(root, 0); err == nil {
		t.Fatal("malformed primary file silently fell back")
	}
	if err := os.Remove(filepath.Join(path, "core_cpus_list")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(path, "core_cpus_list"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := readCoreCPUsAt(root, 0); err == nil {
		t.Fatal("non-missing IO failure silently fell back")
	}
}
