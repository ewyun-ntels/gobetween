//go:build !linux

package udpdispatch

import (
	"fmt"
	"github.com/yyyar/gobetween/config"
)

func NewController(cfg config.Config) (Controller, error) {
	if err := config.ValidateUDPDistributions(cfg); err != nil {
		return nil, err
	}
	for name, server := range cfg.Servers {
		mode, _ := config.UDPDistribution(server)
		if mode == "rr" {
			return nil, fmt.Errorf("server %q: kernel RR requires Linux", name)
		}
	}
	return nil, nil
}
