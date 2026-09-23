package multiprocess

import "github.com/yyyar/gobetween/stats"

const (
	envRole       = "GOBETWEEN_PROCESS_ROLE"
	envWorkerID   = "GOBETWEEN_WORKER_ID"
	envWorkerCPUs = "GOBETWEEN_WORKER_CPUS"
	envIPCFile    = "GOBETWEEN_IPC_FD"
	envReusePort  = "GOBETWEEN_REUSE_PORT"

	roleBootstrap = "bootstrap"
	roleWorker    = "worker"

	messageReady     = "ready"
	messageHeartbeat = "heartbeat"
	messageStats     = "stats"
	messageShutdown  = "shutdown"
	messageStopped   = "stopped"
	messageError     = "error"
)

// Message is the version-one supervisor protocol. Frames are length-prefixed
// JSON so the socket can later carry additional message kinds compatibly.
type Message struct {
	Type     string                 `json:"type"`
	WorkerID int                    `json:"worker_id,omitempty"`
	PID      int                    `json:"pid,omitempty"`
	SentAt   int64                  `json:"sent_at,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Stats    map[string]stats.Stats `json:"stats,omitempty"`
}

// IsBootstrap reports whether this process is the short-lived CPU affinity
// bootstrap which execs the real worker.
func IsBootstrap() bool {
	return processRole() == roleBootstrap
}

// IsWorker reports whether this process is a UDP worker.
func IsWorker() bool {
	return processRole() == roleWorker
}

func processRole() string {
	return getenv(envRole)
}
