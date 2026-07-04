package ring

import (
	"fmt"

	"github.com/grandcat/zeroconf"
)

const (
	AppServiceType   = "_localitas-app._tcp"
	AppServiceDomain = "local."
)

func BroadcastMDNS(port int, name string) (shutdown func(), err error) {
	txt := []string{
		fmt.Sprintf("name=%s", name),
		"plugin_type=homebase-plugin",
		"plugin_for=homebase",
	}
	server, err := zeroconf.Register(name, AppServiceType, AppServiceDomain, port, txt, nil)
	if err != nil {
		return nil, fmt.Errorf("mDNS register: %w", err)
	}
	logger.Info("broadcasting mDNS", "service_type", AppServiceType, "port", port, "name", name)
	return server.Shutdown, nil
}
