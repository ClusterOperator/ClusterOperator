package dto

import "github.com/ClusterOperator/ClusterOperator/pkg/model"

type ClusterTool struct {
	model.ClusterTool
	NodePort string                 `json:"nodePort"`
	Vars     map[string]interface{} `json:"vars"`
}
