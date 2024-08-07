package upgrade

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/service/cluster/adm/phases"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/kobe"
	"io"
)

const (
	upgradeCluster = "92-upgrade-cluster.yml"
)

type UpgradeClusterPhase struct {
	Version string
}

func (upgrade UpgradeClusterPhase) Name() string {
	return "upgradeCluster"
}

func (upgrade UpgradeClusterPhase) Run(b kobe.Interface, writer io.Writer) error {
	if upgrade.Version != "" {
		b.SetVar("kube_upgrade_version", upgrade.Version)
	}
	return phases.RunPlaybookAndGetResult(b, upgradeCluster, "", writer)
}
