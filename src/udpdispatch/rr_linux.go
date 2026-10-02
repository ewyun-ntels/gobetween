//go:build linux

package udpdispatch

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sort"
	"syscall"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/asm"
	"github.com/cilium/ebpf/rlimit"
	"github.com/yyyar/gobetween/config"
	"golang.org/x/sys/unix"
)

type activeWorkers struct {
	Count uint32
	IDs   [MaxWorkers]uint32
}

type rrGroup struct {
	// The anchor keeps the bind group and its BPF policy alive across the
	// last worker's exit. It is never added to the selectable socket map.
	anchor  *net.UDPConn
	bind    string
	address string // resolved once; every worker joins the anchor's endpoint
	workers int
	sockets *ebpf.Map
	active  *ebpf.Map
	counter *ebpf.Map
	program *ebpf.Program
	ready   []bool
	pending map[int]*os.File
}

type controller struct {
	names  []string
	groups map[string]*rrGroup
}

// NewController leaves hash and one-worker configurations completely free of
// BPF resources and privileges. Each rr address has its own shared sequence.
func NewController(cfg config.Config) (Controller, error) {
	if err := config.ValidateUDPDistributions(cfg); err != nil {
		return nil, err
	}
	if cfg.Runtime.WorkerProcesses <= 1 {
		return nil, nil
	}
	var names []string
	binds := make(map[string]string)
	modes := make(map[string]string)
	for name, server := range cfg.Servers {
		mode, _ := config.UDPDistribution(server)
		modes[name] = mode
		if mode == "rr" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil, nil
	}
	if cfg.Runtime.WorkerProcesses > MaxWorkers {
		return nil, fmt.Errorf("rr supports at most %d workers", MaxWorkers)
	}
	// A BPF policy belongs to a bind group, not a server name. Never silently
	// attach rr to a hash server sharing the same resolved local endpoint.
	for name, server := range cfg.Servers {
		addr, err := net.ResolveUDPAddr("udp", server.Bind)
		if err != nil {
			return nil, fmt.Errorf("resolve server %q bind: %w", name, err)
		}
		if addr.Port == 0 && modes[name] == "rr" {
			return nil, fmt.Errorf("server %q: rr requires a fixed UDP bind port", name)
		}
		key := addr.String()
		if other, ok := binds[key]; ok && (modes[name] == "rr" || modes[other] == "rr") {
			return nil, fmt.Errorf("servers %q and %q share an rr bind group", other, name)
		}
		binds[key] = name
	}
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf("prepare kernel RR memory limit: %w", err)
	}
	sort.Strings(names)
	c := &controller{names: names, groups: make(map[string]*rrGroup)}
	for _, name := range names {
		g, err := newRRGroup(cfg.Servers[name].Bind, cfg.Runtime.WorkerProcesses)
		if err != nil {
			_ = c.Close()
			return nil, fmt.Errorf("server %q kernel RR initialization (check BPF capabilities and kernel support): %w", name, err)
		}
		c.groups[name] = g
	}
	return c, nil
}

func newRRGroup(bind string, workers int) (_ *rrGroup, err error) {
	g := &rrGroup{bind: bind, workers: workers, ready: make([]bool, workers), pending: make(map[int]*os.File)}
	address, err := net.ResolveUDPAddr("udp", bind)
	if err != nil {
		return nil, err
	}
	g.address = address.String()
	defer func() {
		if err != nil {
			_ = g.close()
		}
	}()
	g.sockets, err = ebpf.NewMap(&ebpf.MapSpec{Name: "gb_rr_sockets", Type: ebpf.ReusePortSockArray, KeySize: 4, ValueSize: 4, MaxEntries: uint32(workers)})
	if err != nil {
		return nil, err
	}
	// Hash updates replace the whole immutable snapshot under RCU. An array
	// would permit a packet to observe a partially rewritten worker list.
	g.active, err = ebpf.NewMap(&ebpf.MapSpec{Name: "gb_rr_active", Type: ebpf.Hash, KeySize: 4, ValueSize: 4 + 4*MaxWorkers, MaxEntries: 1})
	if err != nil {
		return nil, err
	}
	g.counter, err = ebpf.NewMap(&ebpf.MapSpec{Name: "gb_rr_counter", Type: ebpf.Array, KeySize: 4, ValueSize: 8, MaxEntries: 1})
	if err != nil {
		return nil, err
	}
	if err = g.publish(); err != nil {
		return nil, err
	}
	g.program, err = ebpf.NewProgram(&ebpf.ProgramSpec{
		Name: "gb_udp_rr", Type: ebpf.SkReuseport, License: "GPL",
		Instructions: rrInstructions(g.active.FD(), g.counter.FD(), g.sockets.FD()),
	})
	if err != nil {
		return nil, fmt.Errorf("load RR program (atomic fetch-add required): %w", err)
	}
	g.anchor, err = g.listenSocket(true)
	if err != nil {
		return nil, fmt.Errorf("bind RR control socket: %w", err)
	}
	return g, nil
}

// rrInstructions builds a small native-endian eBPF program in Go; no clang,
// CGo, generated object, kernel patch, or per-packet user-space call is needed.
func rrInstructions(activeFD, counterFD, socketsFD int) asm.Instructions {
	// BPF_ATOMIC | BPF_ADD | BPF_FETCH: return the previous value in R1.
	// A plain XADD followed by a load would not give unique tickets across CPUs.
	fetchAdd := asm.StoreXAdd(asm.R0, asm.R1, asm.DWord)
	fetchAdd.Constant = 1 // BPF_FETCH, ADD is 0x00
	return asm.Instructions{
		asm.Mov.Reg(asm.R6, asm.R1), // preserve sk_reuseport_md
		asm.StoreImm(asm.RFP, -4, 0, asm.Word),
		asm.LoadMapPtr(asm.R1, activeFD),
		asm.Mov.Reg(asm.R2, asm.RFP), asm.Add.Imm(asm.R2, -4),
		asm.FnMapLookupElem.Call(),
		asm.JEq.Imm(asm.R0, 0, "drop"),
		asm.Mov.Reg(asm.R7, asm.R0), // immutable active snapshot
		asm.LoadMem(asm.R8, asm.R7, 0, asm.Word),
		asm.JEq.Imm(asm.R8, 0, "drop"),
		asm.JGT.Imm(asm.R8, MaxWorkers, "drop"),
		asm.LoadMapPtr(asm.R1, counterFD),
		asm.Mov.Reg(asm.R2, asm.RFP), asm.Add.Imm(asm.R2, -4),
		asm.FnMapLookupElem.Call(),
		asm.JEq.Imm(asm.R0, 0, "drop"),
		asm.Mov.Imm(asm.R1, 1), fetchAdd,
		asm.StoreMem(asm.RFP, -16, asm.R1, asm.DWord), // ticket
		asm.Mov.Imm(asm.R9, 0),                        // bounded retry for sockets closing concurrently
		asm.LoadMem(asm.R0, asm.RFP, -16, asm.DWord).WithSymbol("select"),
		asm.Add.Reg(asm.R0, asm.R9), asm.Mod.Reg(asm.R0, asm.R8),
		asm.And.Imm(asm.R0, MaxWorkers-1), // explicit verifier bound
		asm.LSh.Imm(asm.R0, 2), asm.Add.Reg(asm.R0, asm.R7),
		asm.LoadMem(asm.R0, asm.R0, 4, asm.Word),
		asm.StoreMem(asm.RFP, -8, asm.R0, asm.Word),
		asm.Mov.Reg(asm.R1, asm.R6), asm.LoadMapPtr(asm.R2, socketsFD),
		asm.Mov.Reg(asm.R3, asm.RFP), asm.Add.Imm(asm.R3, -8), asm.Mov.Imm(asm.R4, 0),
		asm.FnSkSelectReuseport.Call(),
		asm.JEq.Imm(asm.R0, 0, "pass"),
		asm.Add.Imm(asm.R9, 1),
		asm.JGE.Imm(asm.R9, MaxWorkers, "drop"),
		asm.JLT.Reg(asm.R9, asm.R8, "select"),
		asm.Mov.Imm(asm.R0, 0).WithSymbol("drop"), asm.Return(), // SK_DROP, never implicit hash
		asm.Mov.Imm(asm.R0, 1).WithSymbol("pass"), asm.Return(), // SK_PASS with selected socket
	}
}

func (g *rrGroup) publish() error {
	state := activeWorkers{}
	for id, ready := range g.ready {
		if ready {
			state.IDs[state.Count] = uint32(id)
			state.Count++
		}
	}
	return g.active.Update(uint32(0), &state, ebpf.UpdateAny)
}

func (g *rrGroup) listenSocket(attach bool) (*net.UDPConn, error) {
	lc := net.ListenConfig{Control: func(_, _ string, raw syscall.RawConn) error {
		var err error
		if controlErr := raw.Control(func(fd uintptr) {
			err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
			if err == nil && attach {
				err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_ATTACH_REUSEPORT_EBPF, g.program.FD())
			}
		}); controlErr != nil {
			return controlErr
		}
		return err
	}}
	connection, err := lc.ListenPacket(context.Background(), "udp", g.address)
	if err != nil {
		return nil, err
	}
	udpConn, ok := connection.(*net.UDPConn)
	if !ok {
		_ = connection.Close()
		return nil, fmt.Errorf("unexpected UDP socket type %T", connection)
	}
	return udpConn, nil
}

func (g *rrGroup) listen() (*os.File, error) {
	// Attaching BPF before bind on every socket creates separate groups that
	// cannot join one another. Workers instead inherit the anchor's policy.
	connection, err := g.listenSocket(false)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	return connection.File()
}

func (c *controller) Prepare(workerID int) ([]*os.File, map[string]InheritedListener, error) {
	files := make([]*os.File, 0, len(c.names))
	listeners := make(map[string]InheritedListener, len(c.names))
	for _, name := range c.names {
		g := c.groups[name]
		if workerID <= 0 || workerID > g.workers {
			return nil, nil, fmt.Errorf("invalid rr worker id %d", workerID)
		}
		if g.pending[workerID] != nil || g.ready[workerID-1] {
			return nil, nil, fmt.Errorf("rr worker %d already registered", workerID)
		}
		file, err := g.listen()
		if err != nil {
			_ = c.Remove(workerID)
			return nil, nil, fmt.Errorf("prepare server %q rr socket: %w", name, err)
		}
		g.pending[workerID] = file
		listeners[name] = InheritedListener{FD: 4 + len(files), Bind: g.bind}
		files = append(files, file)
	}
	return files, listeners, nil
}

func (c *controller) Ready(workerID int) error {
	for _, name := range c.names {
		g := c.groups[name]
		if workerID <= 0 || workerID > g.workers {
			return fmt.Errorf("invalid rr worker id %d", workerID)
		}
		file := g.pending[workerID]
		if file == nil {
			return fmt.Errorf("missing pending rr socket for worker %d server %q", workerID, name)
		}
		// File.Fd() can switch a socket to blocking mode. The file description
		// is shared with the already-running child, so use RawConn.Control;
		// otherwise the child's recvfrom can become uncancellable on shutdown.
		raw, err := file.SyscallConn()
		if err != nil {
			return err
		}
		var updateErr error
		if err := raw.Control(func(fd uintptr) { updateErr = g.sockets.Update(uint32(workerID-1), uint32(fd), ebpf.UpdateAny) }); err != nil {
			return err
		}
		if updateErr != nil {
			return fmt.Errorf("register rr worker %d: %w", workerID, updateErr)
		}
		g.ready[workerID-1] = true
		if err := g.publish(); err != nil {
			return fmt.Errorf("publish rr workers: %w", err)
		}
		// The BPF sockarray does not keep the socket alive. The child's socket
		// is now the sole owner, so a crash cannot leave a parent-held black hole.
		_ = file.Close()
		delete(g.pending, workerID)
	}
	return nil
}

func (c *controller) Remove(workerID int) error {
	var result error
	for _, name := range c.names {
		g := c.groups[name]
		if workerID <= 0 || workerID > g.workers {
			return fmt.Errorf("invalid rr worker id %d", workerID)
		}
		g.ready[workerID-1] = false
		if err := g.publish(); err != nil {
			result = errors.Join(result, err)
		}
		if err := g.sockets.Delete(uint32(workerID - 1)); err != nil && !errors.Is(err, ebpf.ErrKeyNotExist) {
			result = errors.Join(result, err)
		}
		if file := g.pending[workerID]; file != nil {
			_ = file.Close()
			delete(g.pending, workerID)
		}
	}
	return result
}

func (g *rrGroup) close() error {
	var result error
	if g.anchor != nil {
		result = errors.Join(result, g.anchor.Close())
	}
	for id, file := range g.pending {
		result = errors.Join(result, file.Close())
		delete(g.pending, id)
	}
	if g.program != nil {
		result = errors.Join(result, g.program.Close())
	}
	for _, m := range []*ebpf.Map{g.sockets, g.active, g.counter} {
		if m != nil {
			result = errors.Join(result, m.Close())
		}
	}
	return result
}

func (c *controller) Close() error {
	var result error
	for _, name := range c.names {
		if g := c.groups[name]; g != nil {
			result = errors.Join(result, g.close())
		}
	}
	return result
}
