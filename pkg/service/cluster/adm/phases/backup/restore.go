package backup

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/service/cluster/adm/phases"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/kobe"
	"io"
)

const (
	restoreCluster = "95-restore-cluster.yml"
)

type RestoreClusterPhase struct {
}

func (restore RestoreClusterPhase) Name() string {
	return "backupCluster"
}

func (restore RestoreClusterPhase) Run(b kobe.Interface, writer io.Writer) error {
	return phases.RunPlaybookAndGetResult(b, restoreCluster, "", writer)
}
