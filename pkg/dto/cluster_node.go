package dto

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/model"
	v1 "k8s.io/api/core/v1"
)

type Node struct {
	model.ClusterNode
	Info v1.Node `json:"info"`
	Ip   string  `json:"ip"`
}

type NodeBatch struct {
	Hosts     []string `json:"hosts"`
	Nodes     []string `json:"nodes"`
	Increase  int      `json:"increase"`
	Operation string   `json:"operation"`
	IsForce   bool     `json:"isForce"`
	StatusID  string   `json:"statusID"`
}

type NodePage struct {
	Items []Node `json:"items"`
	Total int    `json:"total"`
}
