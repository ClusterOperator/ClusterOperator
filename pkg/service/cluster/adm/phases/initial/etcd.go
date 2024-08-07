package initial

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/service/cluster/adm/phases"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/kobe"
	"io"
)

const (
	initEtcd = "06-etcd.yml"
)

type EtcdPhase struct {
	Upgrade bool
}

func (s EtcdPhase) Name() string {
	return "InitEtcd"
}

func (s EtcdPhase) Run(b kobe.Interface, writer io.Writer) error {
	var tag string
	if s.Upgrade {
		tag = "upgrade"
	}
	return phases.RunPlaybookAndGetResult(b, initEtcd, tag, writer)
}
