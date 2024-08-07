package prepare

import (
	"io"

	"github.com/ClusterOperator/ClusterOperator/pkg/service/cluster/adm/phases"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/kobe"
)

const (
	prepareAddWorkerBase = "91-add-worker-01-base.yml"
)

type AddWorkerBaseSystemConfigPhase struct {
}

func (s AddWorkerBaseSystemConfigPhase) Name() string {
	return "BasicConfigSystem"
}

func (s AddWorkerBaseSystemConfigPhase) Run(b kobe.Interface, writer io.Writer) error {
	return phases.RunPlaybookAndGetResult(b, prepareAddWorkerBase, "", writer)
}
