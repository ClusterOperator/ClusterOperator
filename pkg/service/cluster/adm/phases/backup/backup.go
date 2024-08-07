package backup

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/service/cluster/adm/phases"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/kobe"
	"io"
)

const (
	backupCluster = "94-backup-cluster.yml"
)

type BackupClusterPhase struct {
}

func (backup BackupClusterPhase) Name() string {
	return "backupCluster"
}

func (backup BackupClusterPhase) Run(b kobe.Interface, writer io.Writer) error {
	return phases.RunPlaybookAndGetResult(b, backupCluster, "", writer)
}
