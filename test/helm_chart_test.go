package test

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/burntsushi/toml"
	"github.com/yyyar/gobetween/config"
	"gopkg.in/yaml.v3"
)

type chartResource struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Data  map[string]string      `yaml:"data"`
	Spec  map[string]interface{} `yaml:"spec"`
	Rules []struct {
		Resources     []string `yaml:"resources"`
		ResourceNames []string `yaml:"resourceNames"`
		Verbs         []string `yaml:"verbs"`
	} `yaml:"rules"`
}

func renderChart(t *testing.T, overrides ...string) ([]chartResource, string) {
	t.Helper()
	chart, err := filepath.Abs("../charts/gobetween")
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"template", "udp-test", chart, "--namespace", "test-namespace"}
	for _, value := range overrides {
		args = append(args, "--set", value)
	}
	output, err := exec.Command("helm", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("helm template: %v\n%s", err, output)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(output))
	var resources []chartResource
	for {
		var resource chartResource
		if err := decoder.Decode(&resource); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if resource.Kind != "" {
			resources = append(resources, resource)
		}
	}
	return resources, string(output)
}

func chartConfig(t *testing.T, resources []chartResource) config.Config {
	t.Helper()
	for _, resource := range resources {
		if resource.Kind == "ConfigMap" {
			var cfg config.Config
			if _, err := toml.Decode(resource.Data["gobetween.toml"], &cfg); err != nil {
				t.Fatal(err)
			}
			if err := config.ValidateUDPDistributions(cfg); err != nil {
				t.Fatal(err)
			}
			if _, err := config.CPUAllocationPolicy(cfg.Runtime); err != nil {
				t.Fatal(err)
			}
			if cfg.Api.Enabled || cfg.Profiler != nil || len(cfg.Servers) != 1 || cfg.Servers["udpproxy"].Protocol != "udp" {
				t.Fatal("chart must remain UDP-only without API/profiler")
			}
			return cfg
		}
	}
	t.Fatal("no ConfigMap")
	return config.Config{}
}

func TestHelmChart(t *testing.T) {
	if _, err := exec.LookPath("helm"); err != nil {
		if os.Getenv("GOBETWEEN_REQUIRE_HELM") == "1" {
			t.Fatal(err)
		}
		t.Skip("helm not installed")
	}
	t.Run("default_rr_physical", func(t *testing.T) {
		resources, output := renderChart(t)
		cfg := chartConfig(t, resources)
		if cfg.Runtime.WorkerProcesses != 4 || cfg.Runtime.WorkerCPUPolicy != "physical" || cfg.Servers["udpproxy"].Udp.MaxRequests != 1 || cfg.Servers["udpproxy"].Udp.MaxResponses != 0 {
			t.Fatalf("unexpected defaults: %+v", cfg)
		}
		if cfg.Servers["udpproxy"].Discovery.SrvLookupPattern != "_backend._udp.backend-headless.test-namespace.svc.cluster.local." {
			t.Fatal("incorrect SRV name")
		}
		for _, text := range []string{`add: ["BPF"]`, `cpu: "8"`, `memory: "1Gi"`, "readOnlyRootFilesystem: true", "automountServiceAccountToken: false", "hostUsers: true", "type: Recreate"} {
			if !strings.Contains(output, text) {
				t.Errorf("missing %s", text)
			}
		}
		if strings.Count(output, `cpu: "8"`) != 2 || strings.Count(output, `memory: "1Gi"`) != 2 {
			t.Fatal("requests/limits must match")
		}
		if strings.Contains(output, "runAsUser:") || strings.Contains(output, "kind: Role") || strings.Contains(output, "kind: SecurityContextConstraints") || strings.Contains(output, "readinessProbe:") {
			t.Fatal("unexpected UID, permission grant or readiness claim")
		}
	})
	for _, tc := range []struct {
		name     string
		settings []string
	}{
		{"hash", []string{"udp.reusePortDistribution=hash"}},
		{"single_rr", []string{"runtime.workerProcesses=1"}},
	} {
		t.Run(tc.name+"_without_bpf", func(t *testing.T) {
			resources, output := renderChart(t, tc.settings...)
			chartConfig(t, resources)
			if strings.Contains(output, `"BPF"`) {
				t.Fatal("unnecessary BPF capability")
			}
		})
	}
	t.Run("metrics_disabled", func(t *testing.T) {
		resources, output := renderChart(t, "metrics.enabled=false")
		if chartConfig(t, resources).Metrics.Enabled || strings.Contains(output, "startupProbe:") || strings.Contains(output, "livenessProbe:") || strings.Contains(output, "name: metrics") {
			t.Fatal("metrics resources survived disabling")
		}
	})
	t.Run("static_existing_sa_scoped_scc", func(t *testing.T) {
		resources, output := renderChart(t, "discovery.kind=static", "discovery.staticList[0]=127.0.0.1:5000", "serviceAccount.create=false", "serviceAccount.name=existing-sa", "openshift.sccName=approved-udp-rr")
		if chartConfig(t, resources).Servers["udpproxy"].Discovery.StaticList[0] != "127.0.0.1:5000" {
			t.Fatal("incorrect static backend")
		}
		roles := 0
		for _, resource := range resources {
			if resource.Kind == "ServiceAccount" || resource.Kind == "ClusterRole" || resource.Kind == "SecurityContextConstraints" {
				t.Fatal("unexpected cluster/SA mutation")
			}
			if resource.Kind == "Role" {
				roles++
				if len(resource.Rules) != 1 || len(resource.Rules[0].ResourceNames) != 1 || resource.Rules[0].ResourceNames[0] != "approved-udp-rr" || len(resource.Rules[0].Verbs) != 1 || resource.Rules[0].Verbs[0] != "use" {
					t.Fatal("overbroad SCC permissions")
				}
			}
		}
		if roles != 1 || !strings.Contains(output, "namespace: test-namespace") || !strings.Contains(output, "serviceAccountName: existing-sa") {
			t.Fatal("SCC binding missing")
		}
	})
	t.Run("srv_override", func(t *testing.T) {
		resources, _ := renderChart(t, "discovery.srvLookupPattern=_events._udp.udp-headless.backends.svc.cluster.local.")
		if chartConfig(t, resources).Servers["udpproxy"].Discovery.SrvLookupPattern != "_events._udp.udp-headless.backends.svc.cluster.local." {
			t.Fatal("SRV override ignored")
		}
	})
	t.Run("response_timeouts_are_server_values", func(t *testing.T) {
		resources, _ := renderChart(t, "udp.maxRequests=0", "udp.maxResponses=0", "udp.clientIdleTimeout=10s", "udp.backendIdleTimeout=10s")
		server := chartConfig(t, resources).Servers["udpproxy"]
		if server.ClientIdleTimeout == nil || *server.ClientIdleTimeout != "10s" || server.BackendIdleTimeout == nil || *server.BackendIdleTimeout != "10s" {
			t.Fatal("UDP validation requires explicit server idle timeouts")
		}
	})
	for _, value := range []string{"runtime.workerProcesses=0", "runtime.workerProcesses=257", "runtime.workerCPUPolicy=phygical", "resources.cpu=1500m", "resources.cpu=2", "resources.memory=0Gi", "udp.reusePortDistribution=random", "udp.port=80", "terminationGracePeriodSeconds=10", "discovery.kind=static", "serviceAccount.create=false", "hostUsers=false"} {
		t.Run("reject_"+value, func(t *testing.T) {
			chart, _ := filepath.Abs("../charts/gobetween")
			if output, err := exec.Command("helm", "template", "udp-test", chart, "--set", value).CombinedOutput(); err == nil {
				t.Fatalf("invalid setting accepted: %s\n%s", value, output)
			}
		})
	}
}

func TestUDPExampleConfig(t *testing.T) {
	var cfg config.Config
	if _, err := toml.DecodeFile("../config/gobetween-udp.toml", &cfg); err != nil {
		t.Fatal(err)
	}
	if err := config.ValidateUDPDistributions(cfg); err != nil {
		t.Fatal(err)
	}
	policy, err := config.CPUAllocationPolicy(cfg.Runtime)
	if err != nil || policy != "physical" || cfg.Runtime.WorkerProcesses != 4 {
		t.Fatalf("invalid CPU settings: %s %v", policy, err)
	}
	server := cfg.Servers["udpproxy"]
	if len(cfg.Servers) != 1 || server.Protocol != "udp" || server.Udp == nil || server.Udp.ReusePortDistribution != "hash" || server.Udp.MaxRequests != 1 || server.Udp.MaxResponses != 0 || cfg.Api.Enabled || cfg.Profiler != nil {
		t.Fatal("example must be UDP-only one-way hash without BPF/API/profiler")
	}
	if server.Discovery == nil || server.Discovery.Kind != "srv" || server.Discovery.SrvLookupServer != "system" {
		t.Fatal("missing Headless SRV discovery")
	}
}
