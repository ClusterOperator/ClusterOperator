package dto

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/model"
)

type SystemLog struct {
	model.SystemLog
}

type SystemLogCreate struct {
	Name          string `json:"name"`
	Operation     string `json:"operation"`
	OperationInfo string `json:"operationInfo"`
}
