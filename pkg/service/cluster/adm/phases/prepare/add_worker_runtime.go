package prepare

import (
	"io"

	"github.com/ClusterOperator/ClusterOperator/pkg/service/cluster/adm/phases"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/kobe"
)

const (
	prepareAddWorkerContainerRuntime = "91-add-worker-02-runtime.yml"
)

type AddWorkerContainerRuntimePhase struct {
}

func (s AddWorkerContainerRuntimePhase) Name() string {
	return "Install Container Runtime"
}

func (s AddWorkerContainerRuntimePhase) Run(b kobe.Interface, writer io.Writer) error {
	return phases.RunPlaybookAndGetResult(b, prepareAddWorkerContainerRuntime, "", writer)
}
