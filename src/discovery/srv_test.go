package discovery

import (
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
)

func testDNS(t *testing.T, handler dns.HandlerFunc) config.DiscoveryConfig {
	t.Helper()
	socket, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	done := make(chan error, 1)
	server := &dns.Server{PacketConn: socket, Handler: handler, NotifyStartedFunc: func() { close(started) }}
	go func() { done <- server.ActivateAndServe() }()
	select {
	case <-started:
	case err := <-done:
		t.Fatalf("start DNS: %v", err)
	case <-time.After(time.Second):
		t.Fatal("DNS startup timeout")
	}
	t.Cleanup(func() {
		_ = server.Shutdown()
		if err := <-done; err != nil {
			t.Error(err)
		}
	})
	return config.DiscoveryConfig{Kind: "srv", Timeout: "1s", Interval: "10ms", Failpolicy: "keeplast",
		SrvDiscoveryConfig: &config.SrvDiscoveryConfig{SrvLookupServer: socket.LocalAddr().String(), SrvLookupPattern: "_backend._udp.test.local.", SrvDnsProtocol: "udp"}}
}

func TestSRVResponseCodes(t *testing.T) {
	for _, code := range []int{dns.RcodeSuccess, dns.RcodeNameError, dns.RcodeServerFailure, dns.RcodeRefused, dns.RcodeFormatError} {
		t.Run(dns.RcodeToString[code], func(t *testing.T) {
			cfg := testDNS(t, func(w dns.ResponseWriter, r *dns.Msg) {
				response := new(dns.Msg)
				response.SetRcode(r, code)
				_ = w.WriteMsg(response)
			})
			backends, err := srvFetch(cfg)
			valid := code == dns.RcodeSuccess || code == dns.RcodeNameError
			if (err == nil) != valid {
				t.Fatalf("backends=%v err=%v", backends, err)
			}
			if valid && (backends == nil || len(*backends) != 0) {
				t.Fatalf("expected valid empty discovery, got %v", backends)
			}
		})
	}
}

func TestSRVAddressFailure(t *testing.T) {
	cfg := testDNS(t, func(w dns.ResponseWriter, r *dns.Msg) {
		response := new(dns.Msg)
		response.SetReply(r)
		if r.Question[0].Qtype == dns.TypeSRV {
			response.Answer = []dns.RR{&dns.SRV{Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeSRV, Class: dns.ClassINET}, Port: 5000, Target: "backend.test.local."}}
		} else {
			response.Rcode = dns.RcodeServerFailure
		}
		_ = w.WriteMsg(response)
	})
	if backends, err := srvFetch(cfg); err == nil {
		t.Fatalf("address lookup failure became successful discovery: %v", backends)
	}
}

func TestSRVFailurePolicy(t *testing.T) {
	for _, policy := range []string{"keeplast", "setempty"} {
		t.Run(policy, func(t *testing.T) {
			var queries atomic.Int32
			cfg := testDNS(t, func(w dns.ResponseWriter, r *dns.Msg) {
				response := new(dns.Msg)
				response.SetReply(r)
				if queries.Add(1) == 1 {
					response.Answer = []dns.RR{&dns.SRV{Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeSRV, Class: dns.ClassINET}, Port: 5000, Target: "backend.test.local."}}
					response.Extra = []dns.RR{&dns.A{Hdr: dns.RR_Header{Name: "backend.test.local.", Rrtype: dns.TypeA, Class: dns.ClassINET}, A: net.ParseIP("127.0.0.1")}}
				} else {
					response.Rcode = dns.RcodeServerFailure
				}
				_ = w.WriteMsg(response)
			})
			cfg.Failpolicy = policy
			d := NewSrvDiscovery(cfg).(*Discovery)
			d.Start()
			defer d.Stop()
			select {
			case backends := <-d.Discover():
				if len(backends) != 1 {
					t.Fatalf("initial backends=%v", backends)
				}
			case <-time.After(time.Second):
				t.Fatal("missing initial discovery")
			}
			select {
			case backends := <-d.Discover():
				if policy != "setempty" || len(backends) != 0 {
					t.Fatalf("%s: unexpected replacement %v", policy, backends)
				}
			case <-time.After(200 * time.Millisecond):
				if policy == "setempty" {
					t.Fatal("missing empty update")
				}
			}
			if queries.Load() < 2 {
				t.Fatal("failure response was not exercised")
			}
		})
	}
}

func TestDiscoveryStopAfterStaticFetch(t *testing.T) {
	d := &Discovery{cfg: config.DiscoveryConfig{Interval: "0s"}, fetch: func(config.DiscoveryConfig) (*[]core.Backend, error) { return &[]core.Backend{}, nil }}
	d.Start()
	select {
	case <-d.Discover():
	case <-time.After(time.Second):
		t.Fatal("static discovery did not publish")
	}
	done := make(chan struct{})
	go func() { d.Stop(); d.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Stop blocked after static discovery returned")
	}
}
