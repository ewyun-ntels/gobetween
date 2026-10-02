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
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Extra map[string]interface{} `yaml:",inline"`
	Data  map[string]string      `yaml:"data"`
	Spec  map[string]interface{} `yaml:"spec"`
	Rules []struct {
		Resources     []string `yaml:"resources"`
		ResourceNames []string `yaml:"resourceNames"`
		Verbs         []string `yaml:"verbs"`
	} `yaml:"rules"`
}

func renderChart(t *testing.T, overrides ...string) ([]chartResource, string) {
	return renderChartOptions(t, nil, overrides...)
}

func renderChartOptions(t *testing.T, options []string, overrides ...string) ([]chartResource, string) {
	t.Helper()
	chart, err := filepath.Abs("../charts/gobetween")
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"template", "udp-test", chart, "--namespace", "test-namespace"}
	args = append(args, options...)
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
	t.Run("privileged_profile_scoped_scc", func(t *testing.T) {
		resources, _ := renderChartOptions(t, []string{"-f", "../charts/gobetween/values-openshift-privileged.yaml"})
		chartConfig(t, resources)
		sccName := "test-namespace-udp-test-gobetween-privileged"
		counts := map[string]int{}
		for _, resource := range resources {
			counts[resource.Kind]++
			switch resource.Kind {
			case "SecurityContextConstraints":
				if resource.Metadata.Name != sccName || resource.Metadata.Namespace != "" || resource.Extra["allowPrivilegedContainer"] != true || resource.Extra["userNamespaceLevel"] != "AllowHostLevel" || resource.Extra["readOnlyRootFilesystem"] != true {
					t.Fatalf("incorrect dedicated SCC: %+v", resource)
				}
				user := resource.Extra["runAsUser"].(map[string]interface{})
				if user["type"] != "MustRunAs" || user["uid"] != 0 {
					t.Fatalf("SCC must require UID 0: %v", user)
				}
				for _, key := range []string{"allowHostDirVolumePlugin", "allowHostIPC", "allowHostNetwork", "allowHostPID", "allowHostPorts"} {
					if resource.Extra[key] != false {
						t.Fatalf("unexpected host access: %s", key)
					}
				}
				for _, key := range []string{"users", "groups"} {
					if len(resource.Extra[key].([]interface{})) != 0 {
						t.Fatalf("unexpected global SCC grant: %s", key)
					}
				}
			case "Role":
				if resource.Metadata.Namespace != "test-namespace" || len(resource.Rules) != 1 || len(resource.Rules[0].ResourceNames) != 1 || resource.Rules[0].ResourceNames[0] != sccName || len(resource.Rules[0].Verbs) != 1 || resource.Rules[0].Verbs[0] != "use" || len(resource.Rules[0].Resources) != 1 || resource.Rules[0].Resources[0] != "securitycontextconstraints" {
					t.Fatalf("overbroad SCC role: %+v", resource)
				}
			case "RoleBinding":
				subjects := resource.Extra["subjects"].([]interface{})
				if resource.Metadata.Namespace != "test-namespace" || len(subjects) != 1 {
					t.Fatal("binding must target exactly one namespace SA")
				}
				subject := subjects[0].(map[string]interface{})
				if subject["kind"] != "ServiceAccount" || subject["name"] != "udp-test-gobetween" || subject["namespace"] != "test-namespace" {
					t.Fatalf("incorrect SCC subject: %v", subject)
				}
			case "Deployment":
				pod := resource.Spec["template"].(map[string]interface{})["spec"].(map[string]interface{})
				container := pod["containers"].([]interface{})[0].(map[string]interface{})
				for _, sc := range []map[string]interface{}{pod["securityContext"].(map[string]interface{}), container["securityContext"].(map[string]interface{})} {
					if sc["runAsUser"] != 0 || sc["runAsNonRoot"] != false || sc["seccompProfile"] != nil {
						t.Fatalf("inconsistent privileged context: %v", sc)
					}
				}
				sc := container["securityContext"].(map[string]interface{})
				if sc["privileged"] != true || sc["allowPrivilegeEscalation"] != true || sc["readOnlyRootFilesystem"] != true || sc["capabilities"] != nil || pod["hostUsers"] != true {
					t.Fatalf("incorrect privileged container: %v", sc)
				}
			}
		}
		if counts["SecurityContextConstraints"] != 1 || counts["Role"] != 1 || counts["RoleBinding"] != 1 || counts["Deployment"] != 1 || counts["ClusterRole"] != 0 || counts["ClusterRoleBinding"] != 0 {
			t.Fatalf("incorrect resource scope: %v", counts)
		}
	})
	t.Run("privileged_existing_scc", func(t *testing.T) {
		resources, output := renderChart(t, "openshift.privileged=true", "openshift.sccName=privileged", "serviceAccount.create=false", "serviceAccount.name=existing-sa")
		for _, resource := range resources {
			if resource.Kind == "SecurityContextConstraints" || resource.Kind == "ServiceAccount" {
				t.Fatal("existing SCC/SA must not be created or overwritten")
			}
		}
		if !strings.Contains(output, "privileged: true") || !strings.Contains(output, `resourceNames: ["privileged"]`) || !strings.Contains(output, "serviceAccountName: existing-sa") {
			t.Fatal("existing SCC mode not applied")
		}
	})
	t.Run("generated_scc_name_is_namespace_scoped_and_bounded", func(t *testing.T) {
		var names []string
		for _, namespace := range []string{strings.Repeat("a", 63), strings.Repeat("b", 63)} {
			resources, _ := renderChartOptions(t, []string{"--namespace", namespace}, "openshift.privileged=true", "openshift.createSCC=true", "fullnameOverride="+strings.Repeat("c", 54))
			for _, resource := range resources {
				if resource.Kind == "SecurityContextConstraints" {
					if len(resource.Metadata.Name) > 63 {
						t.Fatal("SCC name exceeds DNS label length")
					}
					names = append(names, resource.Metadata.Name)
				}
			}
		}
		if len(names) != 2 || names[0] == names[1] {
			t.Fatal("SCC names collide across namespaces")
		}
	})
	for _, settings := range [][]string{
		{"openshift.createSCC=true"},
		{"openshift.privileged=true"},
		{"openshift.privileged=true", "openshift.createSCC=true", "hostUsers=false"},
		{"containerSecurityContext.privileged=true"},
		{"openshift.privileged=true", "openshift.createSCC=true", "openshift.sccName=privileged"},
		{"openshift.privileged=true", "openshift.createSCC=true", "openshift.sccName=restricted-v2"},
	} {
		t.Run("reject_"+strings.Join(settings, "+"), func(t *testing.T) {
			args := []string{"template", "udp-test", "../charts/gobetween"}
			for _, setting := range settings {
				args = append(args, "--set", setting)
			}
			if output, err := exec.Command("helm", args...).CombinedOutput(); err == nil {
				t.Fatalf("invalid privileged settings accepted: %v\n%s", settings, output)
			}
		})
	}
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
