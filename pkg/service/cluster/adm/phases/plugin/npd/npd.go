package npd

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/service/cluster/adm/phases"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/kobe"
	"io"
)

const (
	npdPlaybook = "12-npd.yml"
)

type NpdPhase struct {
}

func (NpdPhase) Name() string {
	return "Npd"
}

func (c NpdPhase) Run(b kobe.Interface, writer io.Writer) error {
	return phases.RunPlaybookAndGetResult(b, npdPlaybook, "", writer)
}
