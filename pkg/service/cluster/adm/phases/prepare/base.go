package prepare

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/service/cluster/adm/phases"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/kobe"
	"io"
)

const (
	prepareBase = "01-base.yml"
)

type BaseSystemConfigPhase struct {
}

func (s BaseSystemConfigPhase) Name() string {
	return "BasicConfigSystem"
}

func (s BaseSystemConfigPhase) Run(b kobe.Interface, writer io.Writer) error {
	return phases.RunPlaybookAndGetResult(b, prepareBase, "", writer)
}
