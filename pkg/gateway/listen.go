package gateway

import (
	"strconv"

	"github.com/sipeed/picoclaw/pkg/netbind"
)

func openGatewayListeners(host string, port int) (netbind.Plan, netbind.OpenResult, error) {
	plan, err := netbind.BuildPlan(host, netbind.DefaultLoopback)
	if err != nil {
		return netbind.Plan{}, netbind.OpenResult{}, err
	}

	result, err := netbind.OpenPlan(plan, strconv.Itoa(port))
	if err != nil {
		return netbind.Plan{}, netbind.OpenResult{}, err
	}

	return plan, result, nil
}
