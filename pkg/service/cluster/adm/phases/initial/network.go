package initial

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/service/cluster/adm/phases"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/kobe"
	"io"
)

const (
	initNetwork = "09-plugin-network.yml"
)

type NetworkPhase struct {
}

func (NetworkPhase) Name() string {
	return "InitNetwork"
}

func (s NetworkPhase) Run(b kobe.Interface, writer io.Writer) error {
	return phases.RunPlaybookAndGetResult(b, initNetwork, "", writer)
}
