package udp

import "os"

const reusePortEnv = "GOBETWEEN_REUSE_PORT"

func reusePortEnabled() bool {
	return os.Getenv(reusePortEnv) == "1"
}
