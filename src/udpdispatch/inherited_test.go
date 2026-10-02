package udpdispatch

import (
	"github.com/yyyar/gobetween/config"
	"testing"
)

func TestValidateInherited(t *testing.T) {
	t.Setenv(WorkerCountEnv, "4")
	t.Setenv(ListenersEnv, `{"proxy":{"fd":4,"bind":"127.0.0.1:4000"}}`)
	cfg := config.Config{Servers: map[string]config.Server{"proxy": {Protocol: "udp", Bind: "127.0.0.1:4000", Udp: &config.Udp{ReusePortDistribution: "rr"}}}}
	if err := ValidateInherited(cfg); err != nil {
		t.Fatal(err)
	}
	server := cfg.Servers["proxy"]
	server.Bind = "127.0.0.1:4001"
	cfg.Servers["proxy"] = server
	if err := ValidateInherited(cfg); err == nil {
		t.Fatal("accepted mismatched bind")
	}
	t.Setenv(ListenersEnv, "{}")
	if err := ValidateInherited(cfg); err == nil {
		t.Fatal("accepted missing rr descriptor")
	}
	t.Setenv(WorkerCountEnv, "1")
	if err := ValidateInherited(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidDescriptors(t *testing.T) {
	for _, value := range []string{`{"p":{"fd":3,"bind":":4000"}}`, `{"p":{"fd":4,"bind":""}}`, `{"p":{"fd":4,"bind":":4000"},"q":{"fd":4,"bind":":4001"}}`, "invalid"} {
		t.Setenv(ListenersEnv, value)
		if _, err := inheritedDescriptors(); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
}
