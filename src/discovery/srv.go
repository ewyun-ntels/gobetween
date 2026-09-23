package discovery

/**
 * srv.go - SRV record DNS resolve discovery implementation
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
	"github.com/yyyar/gobetween/utils"
)

const (
	srvRetryWaitDuration  = 2 * time.Second
	srvDefaultWaitTimeout = 5 * time.Second
	srvUdpSize            = 4096
)

func NewSrvDiscovery(cfg config.DiscoveryConfig) interface{} {

	d := Discovery{
		opts:  DiscoveryOpts{srvRetryWaitDuration},
		fetch: srvFetch,
		cfg:   cfg,
	}

	return &d
}

/**
 * Create new Discovery with Srv fetch func
 */
func srvFetch(cfg config.DiscoveryConfig) (*[]core.Backend, error) {

	log := logging.For("srvFetch")

	log.Info("Fetching ", cfg.SrvLookupServer, " ", cfg.SrvLookupPattern)

	/* ----- perform query srv  ----- */

	r, err := srvDnsLookup(cfg, cfg.SrvLookupPattern, dns.TypeSRV)
	if err != nil {
		return nil, err
	}

	if len(r.Answer) == 0 {
		log.Warn("Empty response from", cfg.SrvLookupServer, cfg.SrvLookupPattern)
		return &[]core.Backend{}, nil
	}

	/* ----- try to get IPs from additional section ------ */

	hosts := make(map[string]string) // name -> host
	for _, ans := range r.Extra {
		switch record := ans.(type) {
		case *dns.A:
			hosts[record.Header().Name] = record.A.String()
		case *dns.AAAA:
			hosts[record.Header().Name] = fmt.Sprintf("[%s]", record.AAAA.String())
		}
	}

	/* ----- create backend list looking up IP if needed ----- */

	backends := []core.Backend{}
	for _, ans := range r.Answer {
		record, ok := ans.(*dns.SRV)
		if !ok {
			return nil, errors.New("Non-SRV record in SRV answer")
		}

		// If there were no A/AAAA record in additional SRV response,
		// fetch it
		if _, ok := hosts[record.Target]; !ok {
			log.Debug("Fetching ", cfg.SrvLookupServer, " A/AAAA ", record.Target)

			ip, err := srvIPLookup(cfg, record.Target, dns.TypeA)
			if err != nil {
				log.Warn("Error fetching A record for ", record.Target, ": ", err)
			}

			if ip == "" {
				ip, err = srvIPLookup(cfg, record.Target, dns.TypeAAAA)
				if err != nil {
					log.Warn("Error fetching AAAA record for ", record.Target, ": ", err)
				}
			}

			if ip != "" {
				hosts[record.Target] = ip
			} else {
				log.Warn("No IP found for ", record.Target, ", skipping...")
				continue
			}
		}

		// Append new backends
		backends = append(backends, core.Backend{
			Target: core.Target{
				Host: hosts[record.Target],
				Port: fmt.Sprintf("%v", record.Port),
			},
			Priority: int(record.Priority),
			Weight:   int(record.Weight),
			Stats: core.BackendStats{
				Live: true,
			},
			Sni: strings.TrimRight(record.Target, "."),
		})
	}

	return &backends, nil
}

/**
 * Perform DNS Lookup with needed pattern and type
 */
func srvDnsLookup(cfg config.DiscoveryConfig, pattern string, typ uint16) (*dns.Msg, error) {
	timeout := utils.ParseDurationOrDefault(cfg.Timeout, srvDefaultWaitTimeout)
	m := dns.Msg{}
	m.SetQuestion(pattern, typ)
	m.SetEdns0(srvUdpSize, true)

	servers, err := srvDnsServers(cfg.SrvLookupServer)
	if err != nil {
		return nil, err
	}

	network := cfg.SrvDnsProtocol
	if network == "" {
		network = "udp"
	}

	var lastErr error
	for _, server := range servers {
		client := dns.Client{Net: network, Timeout: timeout}
		response, _, exchangeErr := client.Exchange(&m, server)
		if exchangeErr != nil {
			lastErr = exchangeErr
			continue
		}

		// Large headless-service answers may be truncated over UDP. Retry the
		// same query over TCP before treating the result as complete.
		if network == "udp" && response.Truncated {
			client.Net = "tcp"
			response, _, exchangeErr = client.Exchange(&m, server)
			if exchangeErr != nil {
				lastErr = exchangeErr
				continue
			}
		}
		return response, nil
	}

	if lastErr == nil {
		lastErr = errors.New("no DNS servers configured")
	}
	return nil, lastErr
}

func srvDnsServers(configured string) ([]string, error) {
	if configured != "" && configured != "system" {
		return []string{configured}, nil
	}

	resolver, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		return nil, fmt.Errorf("failed to read system DNS configuration: %v", err)
	}
	if len(resolver.Servers) == 0 {
		return nil, errors.New("system DNS configuration contains no nameservers")
	}

	servers := make([]string, 0, len(resolver.Servers))
	for _, server := range resolver.Servers {
		servers = append(servers, net.JoinHostPort(server, resolver.Port))
	}
	return servers, nil
}

/**
 * Perform DNS lookup and extract IP address
 */
func srvIPLookup(cfg config.DiscoveryConfig, pattern string, typ uint16) (string, error) {
	resp, err := srvDnsLookup(cfg, pattern, typ)
	if err != nil {
		return "", err
	}

	if len(resp.Answer) == 0 {
		return "", nil
	}

	switch ans := resp.Answer[0].(type) {
	case *dns.A:
		return ans.A.String(), nil
	case *dns.AAAA:
		return fmt.Sprintf("[%s]", ans.AAAA.String()), nil
	default:
		return "", nil
	}
}
