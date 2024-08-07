package initial

import (
	"io"

	"github.com/ClusterOperator/ClusterOperator/pkg/service/cluster/adm/phases"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/kobe"
)

const (
	initAddWorkerNetwork = "91-add-worker-07-network.yml"
)

type AddWorkerNetworkPhase struct {
}

func (AddWorkerNetworkPhase) Name() string {
	return "InitNetwork"
}

func (s AddWorkerNetworkPhase) Run(b kobe.Interface, writer io.Writer) error {
	return phases.RunPlaybookAndGetResult(b, initAddWorkerNetwork, "", writer)
}
