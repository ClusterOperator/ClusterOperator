package ingress

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/service/cluster/adm/facts"
	"github.com/ClusterOperator/ClusterOperator/pkg/service/cluster/adm/phases"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/kobe"
	"io"
)

const (
	ingressPlaybook = "14-ingress-controller.yml"
)

type ControllerPhase struct {
	IngressControllerType string
}

func (ControllerPhase) Name() string {
	return "IngressController"
}

func (c ControllerPhase) Run(b kobe.Interface, writer io.Writer) error {
	if c.IngressControllerType != "" {
		b.SetVar(facts.IngressControllerTypeFactName, c.IngressControllerType)
	}
	return phases.RunPlaybookAndGetResult(b, ingressPlaybook, "", writer)
}
