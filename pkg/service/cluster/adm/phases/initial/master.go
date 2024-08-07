package initial

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/service/cluster/adm/phases"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/kobe"
	"io"
)

const (
	initMaster = "07-kubernetes-master.yml"
)

type MasterPhase struct {
}

func (MasterPhase) Name() string {
	return "InitEtcd"
}

func (s MasterPhase) Run(b kobe.Interface, writer io.Writer) error {
	return phases.RunPlaybookAndGetResult(b, initMaster, "", writer)
}
