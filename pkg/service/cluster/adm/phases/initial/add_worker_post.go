package initial

import (
	"io"

	"github.com/ClusterOperator/ClusterOperator/pkg/service/cluster/adm/phases"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/kobe"
)

const (
	initAddWorkerPost = "91-add-worker-08-post.yml"
)

type AddWorkerPostPhase struct {
}

func (s AddWorkerPostPhase) Name() string {
	return "Post Init"
}

func (s AddWorkerPostPhase) Run(b kobe.Interface, writer io.Writer) error {
	return phases.RunPlaybookAndGetResult(b, initAddWorkerPost, "", writer)
}
