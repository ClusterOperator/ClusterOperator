package model

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/model/common"
	uuid "github.com/satori/go.uuid"
)

type MultiClusterSyncClusterResourceLog struct {
	common.BaseModel
	ID                           string `json:"-"`
	SourceFile                   string `json:"sourceFile"`
	ResourceName                 string `json:"resourceName"`
	Status                       string `json:"status"`
	Message                      string `json:"message"`
	MultiClusterSyncClusterLogID string `json:"multiClusterSyncLogId"`
}

func (m *MultiClusterSyncClusterResourceLog) BeforeCreate() error {
	m.ID = uuid.NewV4().String()
	return nil
}
