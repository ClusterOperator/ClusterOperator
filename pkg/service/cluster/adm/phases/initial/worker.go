package initial

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/service/cluster/adm/phases"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/kobe"
	"io"
)

const (
	initWorker = "08-kubernetes-worker.yml"
)

type WorkerPhase struct {
}

func (WorkerPhase) Name() string {
	return "InitWorker"
}

func (s WorkerPhase) Run(b kobe.Interface, writer io.Writer) error {
	return phases.RunPlaybookAndGetResult(b, initWorker, "", writer)
}
