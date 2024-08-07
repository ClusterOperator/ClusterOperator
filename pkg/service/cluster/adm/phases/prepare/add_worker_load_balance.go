package prepare

import (
	"io"

	"github.com/ClusterOperator/ClusterOperator/pkg/service/cluster/adm/phases"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/kobe"
)

const (
	prepareAddWorkerLoadBalancer = "91-add-worker-04-load-balancer.yml"
)

type AddWorkerLoadBalancerPhase struct {
}

func (s AddWorkerLoadBalancerPhase) Name() string {
	return "Install Load Balancer"
}

func (s AddWorkerLoadBalancerPhase) Run(b kobe.Interface, writer io.Writer) error {
	return phases.RunPlaybookAndGetResult(b, prepareAddWorkerLoadBalancer, "", writer)
}
