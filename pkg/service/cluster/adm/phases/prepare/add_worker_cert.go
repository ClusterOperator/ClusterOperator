package prepare

import (
	"io"

	"github.com/ClusterOperator/ClusterOperator/pkg/service/cluster/adm/phases"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/kobe"
)

const (
	prepareAddWorkerCertificates = "91-add-worker-05-certificates.yml"
)

type AddWorkerCertificatesPhase struct {
}

func (c AddWorkerCertificatesPhase) Name() string {
	return "GenerateCertificates"
}

func (c AddWorkerCertificatesPhase) Run(b kobe.Interface, writer io.Writer) error {
	return phases.RunPlaybookAndGetResult(b, prepareAddWorkerCertificates, "", writer)
}
